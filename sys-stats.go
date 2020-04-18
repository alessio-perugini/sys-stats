package main

import (
	"flag"
	"fmt"
	g "github.com/soniah/gosnmp"
	"github.com/ziutek/rrd"
	"log"
	"math/big"
	"os"
	"os/signal"
	"time"
)

//Useful NET-SNMP-EXTEND-MIB oids
var oids = map[string]string{
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
	//IF-MIB
	"ifHCInOctets":  ".1.3.6.1.2.1.31.1.1.1.6.2",
	"ifHCOutOctets": ".1.3.6.1.2.1.2.2.1.16.2", //".1.3.6.1.2.1.31.1.1.1.10.2",
	"ifSpeed":       ".1.3.6.1.2.1.2.2.1.5.1",
}

type systemInfo struct {
	cpu     cpuInfo
	memory  memInfo
	network netInfo
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
	//lastUpdate        int64
}

type memInfo struct {
	memTotalReal big.Int
	memAvailReal big.Int
	//lastUpdate   int64
}

type netInfo struct {
	ifHCInOctets  big.Int
	ifHCoutOctets big.Int
	ifSpeed       big.Int
	lastUpdate    int64
}

var (
	hostname    string
	community   string
	port        uint
	versionFlag bool
	interval    string
	rrdDbFile   string

	version = "0.0.0"
	commit  = "commithash"
	sysInfo = systemInfo{}
)

func init() {
	flag.StringVar(&hostname, "host", "localhost", "hostname or ip address")
	flag.StringVar(&community, "community", "public", "community string for snmp")
	flag.UintVar(&port, "port", 161, "port number")
	flag.StringVar(&interval, "interval", "5s", "interval in seconds before send another snmp request")
	flag.StringVar(&rrdDbFile, "rrddb", "test.rrd", "path of rrd database file")
	flag.BoolVar(&versionFlag, "version", false, "output version")
}

func main() {
	flagConfig()
	maxTime, _ := time.ParseDuration(interval)

	//snmp config
	g.Default.Target = hostname
	g.Default.Community = community
	g.Default.Port = uint16(port)

	err := g.Default.Connect() //Open snmp connection
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	}

	defer func() { //Close snmp connection
		if err := g.Default.Conn.Close(); err != nil {
			log.Println(err)
		}
	}()

	rrdCreateChart(uint(maxTime.Seconds()), 5)
	u := rrd.NewUpdater(rrdDbFile)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			rrdInfo()
			fmt.Println(sig)
			rrdCreateGraph()
			os.Exit(1)
		}
	}()

	for {
		tStart := time.Now()
		/*
			getMem()
			if isMaxTimeExpired(tStart, maxTime) {
				continue
			}

			getCPU()
			if isMaxTimeExpired(tStart, maxTime) {
				continue
			}*/

		getNetwork()
		if isMaxTimeExpired(tStart, maxTime) {
			continue
		}

		timeToSleep := maxTime - time.Since(tStart)
		printStats()
		time.Sleep(timeToSleep)
		//18446744073709551615
		rrdUpdate(u, uint32(sysInfo.network.ifHCoutOctets.Int64()))
	}
}

func rrdCreateGraph() {
	// Graph
	graph := rrd.NewGrapher()
	graph.SetTitle("Test")
	graph.SetVLabel("some variable")
	graph.SetSize(800, 300)
	graph.SetWatermark("some watermark")
	graph.Def("outOct", rrdDbFile, "outOct", "AVERAGE")
	//graph.CDef("realOut", "outOct,1048576,/")
	graph.Area("outOct", "#FF0000", "Outbound traffic")
	now := time.Now()

	i, err := graph.SaveGraph("test_rrd1.png", now.Add(-8*time.Minute), now.Add(-30*time.Second))
	fmt.Printf("%+v\n", i)
	if err != nil {
		log.Fatal(err)
	}
}

func rrdInfo() {
	// Info
	inf, err := rrd.Info(rrdDbFile)
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range inf {
		fmt.Printf("%s (%T): %v\n", k, v, v)
	}
}

func rrdUpdate(u *rrd.Updater, args ...interface{}) {
	err := u.Update(time.Now(), args[0])
	if err != nil {
		log.Fatal(err)
	}
}

func rrdCreateChart(step uint, heartbeat int) {
	c := rrd.NewCreator(rrdDbFile, time.Now(), step)
	c.DS("outOct", "COUNTER", 4, 0, uint32(4294967295))
	c.RRA("AVERAGE", 0.5, 2, 16)

	err := c.Create(true)
	if err != nil {
		log.Fatal(err)
	}
}

func isMaxTimeExpired(start time.Time, maxDuration time.Duration) bool {
	elapsedTime := time.Since(start)
	return elapsedTime > maxDuration
}

//Flag config
func flagConfig() {
	appString := fmt.Sprintf("sys-status version %s %s", version, commit)

	flag.Usage = func() { //help flag
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n\nUsage: sys-status [options]\n", appString)
		flag.PrintDefaults()
	}

	flag.Parse()

	if versionFlag { //version flag
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", appString)
		os.Exit(2)
	}

	if v, err := time.ParseDuration(interval); err != nil {
		fmt.Println("Invalid interval format.")
		os.Exit(2)
	} else if v.Seconds() <= 0 {
		fmt.Println("Interval too short it must be at least 1 second long")
		os.Exit(2)
	}

	fmt.Printf("%s\n", appString)
}

//Parse variable of snmp lib to bigint
func parserVariable(v g.SnmpPDU) big.Int {
	return *g.ToBigInt(v.Value)
}

//Obtain cpu statistic
func getCPU() {
	cpuInfoArr := []string{oids["ssCpuIdle"]}
	result, err := g.Default.Get(cpuInfoArr) // // Send snmp get and retrieve values up to g.MAX_OIDS
	if err != nil {
		log.Fatalf("Get() err: %v", err)
	}

	sysInfo = systemInfo{
		cpu: cpuInfo{ //parse variable and populate cpu struct
			ssCpuIdle: parserVariable(result.Variables[0]),
		},
		memory:  sysInfo.memory,
		network: sysInfo.network,
	}
}

//Obtain memory statistic
func getMem() {
	memInfoArr := []string{oids["memTotalReal"], oids["memAvailReal"]}
	result, err := g.Default.Get(memInfoArr) // Send snmp get and retrieve values up to g.MAX_OIDS
	if err != nil {
		log.Fatalf("Get() err: %v", err)
	}

	sysInfo = systemInfo{
		cpu: sysInfo.cpu,
		memory: memInfo{ //parse variable and populate memory struct
			memTotalReal: parserVariable(result.Variables[0]),
			memAvailReal: parserVariable(result.Variables[1]),
		},
		network: sysInfo.network,
	}
}

func getNetwork() {
	netInfoArr := []string{oids["ifHCInOctets"], oids["ifHCOutOctets"], oids["ifSpeed"]}
	result, err := g.Default.Get(netInfoArr) // Send snmp get and retrieve values up to g.MAX_OIDS

	if err != nil {
		log.Fatalf("Get() err: %v", err)
	}

	newIn := parserVariable(result.Variables[0])
	newOut := parserVariable(result.Variables[1])
	newSpeed := parserVariable(result.Variables[2])

	sysInfo = systemInfo{
		cpu:    sysInfo.cpu,
		memory: sysInfo.memory,
		network: netInfo{
			ifHCInOctets:  newIn,
			ifHCoutOctets: newOut,
			ifSpeed:       newSpeed,
			lastUpdate:    time.Now().UnixNano(),
		},
	}

}

//print info about cpu and ram
func printStats() {
	cpuLoad := new(big.Int).Sub(big.NewInt(100), &sysInfo.cpu.ssCpuIdle) //Max load (100) - IdleLoad

	KBtoGB := big.NewFloat(float64(1024 * 1024)) //Used for conversion from KB to GB
	memAvailGB := new(big.Float).Quo(bIntToBFloat(sysInfo.memory.memAvailReal), KBtoGB)
	memTotalGB := new(big.Float).Quo(bIntToBFloat(sysInfo.memory.memTotalReal), KBtoGB)

	fmt.Printf("Memory available: %.2f/%.2f GB\n", memAvailGB, memTotalGB)
	fmt.Printf("CPU usage:        %d%s\n", cpuLoad, "%")
}

//bigint to bigfloat
func bIntToBFloat(v big.Int) *big.Float {
	return new(big.Float).SetInt(&v)
}
