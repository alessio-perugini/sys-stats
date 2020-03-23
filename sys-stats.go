package main


//TODO aggiungere flag, commentare e add readme
import (
	"flag"
	"fmt"
	g "github.com/soniah/gosnmp"
	"log"
	"math/big"
	"os"
)

//Usefull NET-SNMP-EXTEND-MIB oids
var OIDS = map[string]string{
	//UCD-SNMP-MIB
	"ssIndex":           ".1.3.6.1.4.1.2021.11.1.0",
	"ssErrorName":       ".1.3.6.1.4.1.2021.11.2.0",
	"ssSwapIn":          ".1.3.6.1.4.1.2021.11.3.0",
	"ssSwapOut":         ".1.3.6.1.4.1.2021.11.4.0",
	"ssCpuIdle":         ".1.3.6.1.4.1.2021.11.11.0",
	"ssCpuRawUser":      ".1.3.6.1.4.1.2021.11.50.0",
	"ssCpuRawNice":      ".1.3.6.1.4.1.2021.11.51.0",
	"ssCpuRawSystem":    ".1.3.6.1.4.1.2021.11.52.0",
	"ssCpuRawIdle":      ".1.3.6.1.4.1.2021.11.53.0",
	"ssCpuRawWait":      ".1.3.6.1.4.1.2021.11.54.0",
	"ssCpuRawKernel":    ".1.3.6.1.4.1.2021.11.55.0",
	"ssCpuRawInterrupt": ".1.3.6.1.4.1.2021.11.56.0",
	"ssIORawSent":       ".1.3.6.1.4.1.2021.11.57.0",
	"ssIORawReceived":   ".1.3.6.1.4.1.2021.11.58.0",
	"ssRawInterrupts":   ".1.3.6.1.4.1.2021.11.59.0",
	"ssRawContexts":     ".1.3.6.1.4.1.2021.11.60.0",
	"ssCpuRawSoftIRQ":   ".1.3.6.1.4.1.2021.11.61.0",
	"ssRawSwapIn":       ".1.3.6.1.4.1.2021.11.62.0",
	"ssRawSwapOut":      ".1.3.6.1.4.1.2021.11.63.0",
	"ssCpuRawSteal":     ".1.3.6.1.4.1.2021.11.64.0",
	"ssCpuRawGuest":     ".1.3.6.1.4.1.2021.11.65.0",
	"ssCpuRawGuestNice": ".1.3.6.1.4.1.2021.11.66.0",
	"memTotalReal":      ".1.3.6.1.4.1.2021.4.5.0",
	"memAvailReal":      ".1.3.6.1.4.1.2021.4.6.0",
}

type systemInfo struct {
	cpu    cpuInfo
	memory memInfo
}

type cpuInfo struct {
	ssCpuIdle         big.Int
	ssCpuRawUser      big.Int
	ssCpuRawNice      big.Int
	ssCpuRawSystem    big.Int
	ssCpuRawIdle      big.Int
	ssCpuRawWait      big.Int
	ssCpuRawKernel    big.Int
	ssCpuRawInterrupt big.Int
	ssCpuRawSteal     big.Int
	ssCpuRawGuest     big.Int
	ssCpuRawGuestNice big.Int
}

type memInfo struct {
	memTotalReal big.Int
	memAvailReal big.Int
}

var (
	hostname    string
	community   string
	port        uint
	logsFlag    bool
	versionFlag bool

	version = "0.0.0"
	commit  = "commithash"
	sysInfo = systemInfo{}
)

func init() {
	flag.StringVar(&hostname, "host", "localhost", "hostname or ip address")
	flag.StringVar(&community, "community", "public", "community string for snmp")
	flag.UintVar(&port, "port", 161, "port number")
	flag.BoolVar(&logsFlag, "logs", false, "enable logs")
	flag.BoolVar(&versionFlag, "version", false, "output version")
}

func main() {
	appString := fmt.Sprintf("sys-status version %s %s", version, commit)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n\nUsage: sys-status [options]\n", appString)
		flag.PrintDefaults()
	}

	flag.Parse()

	if versionFlag {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", appString)
		os.Exit(2)
	}

	fmt.Printf("%s\n", appString)

	g.Default.Target = hostname
	g.Default.Community = community
	g.Default.Port = uint16(port)

	err := g.Default.Connect()
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	}

	defer g.Default.Conn.Close()

	getMem()
	getCpu()
	printStats()
}

func parserVariable(v g.SnmpPDU) big.Int {
	return *g.ToBigInt(v.Value)
}

func getCpu() {
	cpuInfoArr := []string{OIDS["ssCpuIdle"]}
	result, err := g.Default.Get(cpuInfoArr) // Get() accepts up to g.MAX_OIDS
	if err != nil {
		log.Fatalf("Get() err: %v", err)
	}

	sysInfo = systemInfo{
		cpu: cpuInfo{
			ssCpuIdle: parserVariable(result.Variables[0]),
		},
		memory: sysInfo.memory,
	}
}

func getMem() {
	memInfoArr := []string{OIDS["memTotalReal"], OIDS["memAvailReal"]}
	result, err := g.Default.Get(memInfoArr) // Get() accepts up to g.MAX_OIDS
	if err != nil {
		log.Fatalf("Get() err: %v", err)
	}

	sysInfo = systemInfo{
		memory: memInfo{
			memTotalReal: parserVariable(result.Variables[0]),
			memAvailReal: parserVariable(result.Variables[1]),
		},
	}
}

func getDisk() {

}

func printStats() {
	cpuLoad := new(big.Int).Sub(big.NewInt(100), &sysInfo.cpu.ssCpuIdle)

	KBtoGB := big.NewFloat(float64(1024 * 1024)) //Used for conversion from KB to GB
	memAvailGB := new(big.Float).Quo(bIntToBFloat(sysInfo.memory.memAvailReal), KBtoGB)
	memTotalGB := new(big.Float).Quo(bIntToBFloat(sysInfo.memory.memTotalReal), KBtoGB)

	fmt.Printf("Memory available: %.2f/%.2f GB\n", memAvailGB, memTotalGB)
	fmt.Printf("CPU usage:        %d%s", cpuLoad, "%")
}

//bigint to bigfloat
func bIntToBFloat(v big.Int) *big.Float {
	return new(big.Float).SetInt(&v)
}
