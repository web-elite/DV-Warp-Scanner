package main

import (
	"fmt"
	"os"
	"strings"
)

func managementMenu() {
	for {
		fmt.Println()
		fmt.Println(fmtStr("DV Warp Scanner · Management", BLUE, true))
		fmt.Printf("  %s  %s\n", fmtStr("0", GREEN, true), "Start scan")
		fmt.Printf("  %s  %s  %s\n", fmtStr("1", GREEN, true), "Xray-core manager", fmtStr("("+activeXrayVersion()+")", ORANGE, false))
		fmt.Printf("  %s  %s\n", fmtStr("2", GREEN, true), "About")
		fmt.Printf("  %s  %s\n", fmtStr("3", GREEN, true), "Exit")
		switch readMenuChoice("\nSelect an option: ") {
		case "0":
			startScan()
		case "1":
			xrayManagerMenu()
		case "2":
			fmt.Printf("\nScanner version: %s\nXray-core version: %s\n", VERSION, activeXrayVersion())
			fmt.Println("Xray-core releases: https://github.com/XTLS/Xray-core/releases")
		case "3":
			fmt.Println("Goodbye.")
			os.Exit(0)
		default:
			failMessage("Invalid option. Choose 0, 1, 2 or 3.")
		}
	}
}

func readMenuChoice(promptText string) string {
	fmt.Print(promptText)
	var value string
	_, _ = fmt.Scanln(&value)
	return strings.TrimSpace(value)
}
