package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var income float64

type ledger struct {
	processID    string
	companyName  string
	stocksBought int64
	stocksPrice  float64
}

type stock struct { //structure for stock
	companyName    string  //name of the company
	price          float64 //price of the stock
	totalCount     int64   //total count available in the market
	countAvailable int64   //number of stocks available
}

func New(companyName string, price float64, countAvailable int64) *stock {

	//function for creating a new stock object
	st := stock{companyName, price, countAvailable, countAvailable}
	return &st
}

func NewLedger(processID string, companyName string, stocksBought int64, price float64) *ledger {
	record := ledger{processID, companyName, stocksBought, price}
	return &record
}

func View(comp *stock) {

	//function for viewing the stock details
	fmt.Println("\n\nCompany details")
	fmt.Println("Name of the company:", comp.companyName)
	fmt.Println("Stock offering: ", comp.countAvailable)
	fmt.Println("Stock price: ", comp.price)
}

func connection(offerings map[string]*stock, ledgerKeeping map[time.Time]*ledger) {

	var data []byte

	s, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49200")
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenUDP("udp4", s)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer connection.Close()
	buffer := make([]byte, 1024)

	for {
		size, addr, err := connection.ReadFromUDP(buffer) //getting the processID
		dataID := string(buffer[0 : size-1])
		dataID = strings.TrimSpace(string(dataID))
		processNo := dataID

		n, addr, err := connection.ReadFromUDP(buffer) //getting information about what the client wants to view
		dataFrom := string(buffer[0 : n-1])
		dataFrom = strings.TrimSpace(string(dataFrom))
		if dataFrom == "ALL" {

			for keys := range offerings {
				data = []byte(offerings[keys].companyName + " as " + keys)
				_, err = connection.WriteToUDP(data, addr) //sending all the details
			}
			data = []byte("EOF")
			_, err = connection.WriteToUDP(data, addr)
			continue

		} else {
			cName := dataFrom
			_, present := offerings[dataFrom]
			if present {
				price := strconv.FormatFloat(offerings[dataFrom].price, 'g', 10, 64) //convert price to string format
				temp := int(offerings[dataFrom].countAvailable)
				available := strconv.Itoa(temp)
				dataTemp := available
				data = []byte(dataTemp)
				_, err = connection.WriteToUDP(data, addr) //sending the number of stocks available to the client
				if err != nil {
					fmt.Println(err)
					return
				}
				dataPrice := []byte(price)
				_, err = connection.WriteToUDP(dataPrice, addr) //sending the price to the client
				if err != nil {
					fmt.Println(err)
					return
				}
				n, addr, err := connection.ReadFromUDP(buffer) //getting the number of stocks that the client wants
				requirement := string(buffer[0 : n-1])
				requirement = strings.TrimSpace(string(requirement))
				required, err := strconv.ParseInt(requirement, 10, 64)
				if err != nil {
					fmt.Println(err)
				}
				if required <= offerings[dataFrom].countAvailable {
					ledgerKeeping[time.Now()] = NewLedger(processNo, cName, required, offerings[cName].price)
					offerings[dataFrom].countAvailable = offerings[dataFrom].countAvailable - required
					_, err = connection.WriteToUDP([]byte("SUCCESS"), addr) //sending the success message
					if err != nil {
						fmt.Println(err)
						return
					}
					income += float64(required) * offerings[dataFrom].price
				} else {
					_, err = connection.WriteToUDP([]byte("FAILED"), addr) //sending the failure message
					if err != nil {
						fmt.Println(err)
						return
					}

				}

			} else {
				//if offering is not available
				data = []byte("INVALID")
				_, err = connection.WriteToUDP(data, addr)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}

}

func main() {

	income = 0
	//for getting the input
	var in *bufio.Reader
	in = bufio.NewReader(os.Stdin)

	var ledgerKeeping map[time.Time]*ledger
	ledgerKeeping = make(map[time.Time]*ledger)

	//all the offerings
	var offerings map[string]*stock
	offerings = make(map[string]*stock)

	//initial offerings
	offerings["Reliance"] = New("Reliance Industries Ltd", 92, 100)
	offerings["Aston Martin"] = New("Aston Martin Lagonda Private Holdings plc", 12, 1000)
	offerings["ICICI"] = New("ICICI Bank", 40, 500)

	var choice int64
	var cName string
	var priceOffering float64
	var stocksOffered int64

	go connection(offerings, ledgerKeeping)
	for true {
		fmt.Println("\nCentral system")
		fmt.Println("\nEnter the operation you would like to perform: ")
		fmt.Println("\n1.Add\n2.Remove\n3.View companies\n4.View ledger\n5.Print income\n6.Close")
		fmt.Scanln(&choice)
		if choice == 1 {

			//to add new offerings to the existing list
			fmt.Println("Enter the display name of the company: ")
			fmt.Scanln(&cName)

			fmt.Println("Enter the full name of the company: ")
			companyName, err := in.ReadString('\n')
			if err != nil {
				fmt.Println("\nInput error")
			}

			fmt.Println("Enter the number of stocks you are offering: ")
			fmt.Scanln(&stocksOffered)

			fmt.Println("Enter the price of the stock: ")
			fmt.Scanln(&priceOffering)

			offerings[cName] = New(companyName, priceOffering, stocksOffered)
		} else if choice == 2 {

			//remove an offering from the existing list
			fmt.Println("Enter the display name of the company: ")
			fmt.Scanln(&cName)
			_, present := offerings[cName]
			if present {

				//check if the offering exists then delete
				View(offerings[cName])
				delete(offerings, cName)
			} else {

				//if offering is not available
				fmt.Println("\nThe Company doesn't exist")
			}
		} else if choice == 3 {

			//to view the details regarding a company
			fmt.Println("Enter the display name of the company: ")
			fmt.Scanln(&cName)

			View(offerings[cName])
		} else if choice == 4 {

			for keys := range ledgerKeeping {
				fmt.Printf("\n")
				fmt.Printf(ledgerKeeping[keys].processID+" bought %d", ledgerKeeping[keys].stocksBought)
				fmt.Printf(" stocks of " + ledgerKeeping[keys].companyName + " at " + keys.String() + " for ")
				fmt.Printf("%f", ledgerKeeping[keys].stocksPrice)
			}

		} else if choice == 5 {
			fmt.Println("\nIncome: ", income)

		} else if choice == 6 {

			//to close the exchange market
			fmt.Println("The stock market is closing for the day.")
			time.Sleep(2 * time.Second)
			os.Exit(0)
		} else {
			fmt.Println("Invalid input")
		}
	}
}
