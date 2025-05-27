package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
)

var (
	reportDir = "Reporter"
	jtlDir    = "Jtl"
	debugDir  = "Debug"
	logFile   = filepath.Join(debugDir, "pmeter.log")
)

func ensureDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("无法创建目录 %s: %v\n", dir, err)
		}
	}
}

func writeLog(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	ensureDir(debugDir)

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("无法打开日志文件:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, msg)
	if _, err := file.WriteString(logEntry); err != nil {
		log.Println("写入日志失败:", err)
	}
}

func listJMXFiles() []string {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		writeLog("读取目录失败: " + err.Error())
		fmt.Println("无法读取当前目录")
		os.Exit(1)
	}

	var jmxFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jmx") {
			jmxFiles = append(jmxFiles, file.Name())
		}
	}

	if len(jmxFiles) == 0 {
		writeLog("未找到 .jmx 文件，程序退出。")
		fmt.Println("当前目录下没有找到 .jmx 文件。")
		os.Exit(1)
	}

	fmt.Println("可用的 .jmx 文件：")
	for idx, file := range jmxFiles {
		fmt.Printf("%d. %s\n", idx+1, file)
	}

	return jmxFiles
}

func selectJMXFile(jmxFiles []string) string {
	fmt.Print("请选择要运行的测试文件编号：")
	var choice string
	fmt.Scanln(&choice)

	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(jmxFiles) {
		writeLog(fmt.Sprintf("无效的文件编号输入：%s，程序退出。", choice))
		fmt.Println("输入无效，退出。")
		os.Exit(1)
	}

	return jmxFiles[index-1]
}

func getTimestamp() string {
	return time.Now().Format("20060102_150405")
}

func getResultFilename() string {
	fmt.Print("请输入结果文件名（例如 result_200 或 result_200.jtl，回车则使用时间戳）：")
	var name string
	fmt.Scanln(&name)
	name = strings.TrimSpace(name)

	if name == "" {
		timestamp := getTimestamp()
		name = fmt.Sprintf("result_%s.jtl", timestamp)
		fmt.Printf("已自动生成结果文件名：%s\n", name)
	} else if !strings.HasSuffix(name, ".jtl") {
		name += ".jtl"
	}

	return filepath.Join(jtlDir, name)
}

func getReportFolder() string {
	fmt.Print("请输入输出报告文件夹名称（例如 report_fold，回车则使用时间戳）：")
	var folder string
	fmt.Scanln(&folder)
	folder = strings.TrimSpace(folder)

	if folder == "" {
		timestamp := getTimestamp()
		folder = fmt.Sprintf("report_%s", timestamp)
		fmt.Printf("已自动生成报告文件夹：%s\n", folder)
	}

	fullPath := filepath.Join(reportDir, folder)
	ensureDir(fullPath)
	fmt.Printf("报告文件夹路径：%s\n", fullPath)

	return fullPath
}

func getRDebugFilename() string {
	fmt.Print("请输入DEBUG日志文件名（例如 XXXXX_debug.log，回车则使用时间戳）：")
	var name string
	fmt.Scanln(&name)
	name = strings.TrimSpace(name)

	if name == "" {
		timestamp := getTimestamp()
		name = fmt.Sprintf("Debug_%s.log", timestamp)
		fmt.Printf("已自动生成Debug日志文件名：%s\n", name)
	} else if !strings.HasSuffix(name, ".log") {
		name += ".log"
	}

	return filepath.Join(debugDir, name)
}

// 使用XPath解析线程数
func parseThreadCount(jmxFile string) int {
	data, err := ioutil.ReadFile(jmxFile)
	if err != nil {
		writeLog(fmt.Sprintf("读取JMX文件失败: %v", err))
		return 0
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(data)))
	if err != nil {
		writeLog(fmt.Sprintf("解析XML失败: %v", err))
		return 0
	}

	node := xmlquery.FindOne(doc, "//intProp[@name='ThreadGroup.num_threads']")
	if node == nil {
		writeLog("未找到线程数配置")
		return 0
	}

	count, err := strconv.Atoi(node.InnerText())
	if err != nil {
		writeLog(fmt.Sprintf("转换线程数失败: %v", err))
		return 0
	}

	return count
}

type Statistics struct {
	Total struct {
		SampleCount int     `json:"sampleCount"`
		ErrorPct    float64 `json:"errorPct"`
		MeanResTime float64 `json:"meanResTime"`
		Throughput  float64 `json:"throughput"`
		Errorcount  int     `json:"errorCount"`
	} `json:"Total"`
}

func parseStatistics(reportFolder string) (map[string]interface{}, error) {
	statFile := filepath.Join(reportFolder, "statistics.json")
	data, err := ioutil.ReadFile(statFile)
	if err != nil {
		return nil, err
	}

	var stats Statistics
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"sampleCount": stats.Total.SampleCount,
		"errorPct":    stats.Total.ErrorPct,
		"meanResTime": stats.Total.MeanResTime,
		"throughput":  stats.Total.Throughput,
		"Errorcount":  stats.Total.Errorcount,
	}

	return result, nil
}

func runJMeter(jmxFile, resultFile, reportFolder, debugFile string) {
	cmd := exec.Command("jmeter", "-n", "-t", jmxFile, "-l", resultFile, "-e", "-o", reportFolder, "-j", debugFile)
	startTime := time.Now()
	cmdStr := strings.Join(cmd.Args, " ")
	writeLog("开始执行命令：")
	writeLog(cmdStr)

	fmt.Println("\n执行命令：", cmdStr)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	endTime := time.Now()
	duration := endTime.Sub(startTime).Seconds()

	if err != nil {
		writeLog(fmt.Sprintf("命令执行失败，耗时 %.2f 秒。错误: %v", duration, err))
		fmt.Println("命令执行失败，请检查日志文件 pmeter.log。")
		return
	}

	writeLog(fmt.Sprintf("命令执行成功，耗时 %.2f 秒。", duration))

	threadCount := parseThreadCount(jmxFile)
	stats, err := parseStatistics(reportFolder)

	if err == nil {
		writeLog("====== 测试统计信息 ======")
		writeLog(fmt.Sprintf("线程数：%d", threadCount))
		writeLog(fmt.Sprintf("报错率：%.2f%%", stats["errorPct"].(float64)))
		writeLog(fmt.Sprintf("吞吐量：%.2f", stats["throughput"].(float64)))
		writeLog(fmt.Sprintf("响应时间：%.2f ms", stats["meanResTime"].(float64)))
		writeLog(fmt.Sprintf("总请求数：%d", stats["sampleCount"].(int)))
		writeLog(fmt.Sprintf("总错误数：%d", stats["Errorcount"].(int)))
		writeLog("==========================")
	} else {
		writeLog(fmt.Sprintf("解析统计数据失败: %v", err))
	}
}

func main() {
	ensureDir(reportDir)
	ensureDir(jtlDir)
	ensureDir(debugDir)

	jmxFiles := listJMXFiles()
	jmxFile := selectJMXFile(jmxFiles)
	resultFile := getResultFilename()
	reportFolder := getReportFolder()
	debugFile := getRDebugFilename()
	runJMeter(jmxFile, resultFile, reportFolder, debugFile)
}
