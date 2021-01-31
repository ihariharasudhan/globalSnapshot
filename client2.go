package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type State struct {
	VectorTime [3]int
}

type Request struct {
	ProcessID   int64
	CompanyName string
	StockCount  int64
	StockPrice  float64
}

type Clock struct {
	ProcessID  int64
	VectorTime [3]int
}

type ledger struct {
	processID    string
	companyName  string
	stocksBought int64
	stockPrice   float64
}

type stock struct { //structure for stock
	companyName    string  //name of the company
	price          float64 //price of the stock
	countAvailable int64   //number of stocks available
}

func New(companyName string, price float64, countAvailable int64) *stock {

	//function for creating a new stock object
	st := stock{companyName, price, countAvailable}
	return &st
}

func viewState(st *State) {
	fmt.Println("-", st.VectorTime)
}

func View(comp *stock) {

	//function for viewing the stock details
	fmt.Println("\n\nCompany details")
	fmt.Println("Name of the company:", comp.companyName)
	fmt.Println("Stock offering: ", comp.countAvailable)
	fmt.Println("Stock price: ", comp.price)
}

var offerings map[string]*stock
var income float64
var ledgerKeeping map[[3]int]*ledger
var localTime Clock
var allStates map[int]*State
var stateID int
var stateRecorded, channel0To1, channel2To1 bool

func NewLedger(processID string, companyName string, stocksBought int64, price float64) *ledger {
	record := ledger{processID, companyName, stocksBought, price}
	return &record
}

func newState() *State {
	st := State{localTime.VectorTime}
	return &st

}

func syncTimeReceiver() {

	var tempTime Clock
	var companyWant Request
	Found := 0

	s, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49202")
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
		size, addr, _ := connection.ReadFromUDP(buffer)
		timeGot := buffer[0:size]
		json.Unmarshal(timeGot, &tempTime)
		for system := 0; system < 3; system++ {
			if localTime.VectorTime[system] < tempTime.VectorTime[system] {
				localTime.VectorTime[system] = tempTime.VectorTime[system]
			}
		}
		n, addr, _ := connection.ReadFromUDP(buffer)
		companyDetails := buffer[0:n]
		json.Unmarshal(companyDetails, &companyWant)

		localTime.VectorTime[localTime.ProcessID] += 1

		cName := companyWant.CompanyName
		pId := strconv.Itoa(int(companyWant.ProcessID))

		for keys := range offerings {
			if offerings[keys].companyName == cName {
				if companyWant.StockCount <= offerings[keys].countAvailable {
					if companyWant.StockPrice > offerings[keys].price {
						offerings[keys].countAvailable -= companyWant.StockCount
						ledgerKeeping[localTime.VectorTime] = NewLedger(pId, cName, companyWant.StockCount, companyWant.StockPrice)
						_, err := connection.WriteToUDP([]byte("SUCCESS"), addr)
						if err != nil {
							fmt.Println(err)
						}
						income = income + float64(companyWant.StockCount)*companyWant.StockPrice
						Found = 1
						break
					}
				}
			}

		}
		if Found == 0 {
			_, err := connection.WriteToUDP([]byte("FAILED"), addr)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			Found = 0
		}
		localTime.VectorTime[localTime.ProcessID] += 1
		timeSend, _ := json.Marshal(localTime)
		_, err = connection.WriteToUDP(timeSend, addr)
	}

}

func sendMarker() {

	allStates[stateID] = newState() //saving the state of the process
	stateID = stateID + 1
	stateRecorded = true

	setUp1, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49204")
	connection1, err := net.DialUDP("udp4", nil, setUp1)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection1.Close()
	_, err = connection1.Write([]byte("1"))

	setUp2, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49206")
	connection2, err := net.DialUDP("udp4", nil, setUp2)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection2.Close()
	_, err = connection2.Write([]byte("1"))

}

func receiveMarker() {

	s, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49205")
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
		if channel0To1 == true && channel2To1 == true {
			channel0To1 = false
			channel2To1 = false
			stateRecorded = false
		}

		size, _, _ := connection.ReadFromUDP(buffer)
		gotRequestFrom := string(buffer[0:size])
		if stateRecorded != true {

			sendMarker() //sending the request to another process
		}
		if gotRequestFrom == "0" && channel0To1 == false {
			//set channel state if necessary
			channel0To1 = true
		} else if gotRequestFrom == "2" && channel2To1 == false {
			//set channel state if necessary
			channel2To1 = true
		}

	}

}

func syncTimeSender(IP string) {

	timeSend, _ := json.Marshal(localTime)

	setUp, err := net.ResolveUDPAddr("udp4", IP)
	connection, err := net.DialUDP("udp4", nil, setUp)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection.Close()
	_, err = connection.Write(timeSend)

}

func sendRequest(IP string) {

	var tempTime Clock
	var requestSend Request
	var in *bufio.Reader
	in = bufio.NewReader(os.Stdin)

	setUp, err := net.ResolveUDPAddr("udp4", IP)
	connection, err := net.DialUDP("udp4", nil, setUp)
	if err != nil {
		fmt.Println(err)
		return
	}

	requestSend.ProcessID = localTime.ProcessID
	fmt.Println("\nEnter the name of the company: ")
	cName, _ := in.ReadString('\n')
	requestSend.CompanyName = cName
	fmt.Println("\nEnter the number of stocks you would like to buy: ")
	fmt.Scanln(&requestSend.StockCount)
	fmt.Println("\nEnter the bid amount per stock: ")
	fmt.Scanln(&requestSend.StockPrice)

	sendData, _ := json.Marshal(requestSend)
	_, err = connection.Write(sendData)

	buffer := make([]byte, 1024)
	n, _, err := connection.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println(err)
		return
	}
	reply := string(buffer[0:n])
	reply = strings.TrimSpace(string(reply))
	if reply == "SUCCESS" {
		ledgerKeeping[localTime.VectorTime] = NewLedger("1", cName, requestSend.StockCount, requestSend.StockPrice)
		offerings[cName].countAvailable += requestSend.StockCount
		income = income - float64(requestSend.StockCount)*requestSend.StockPrice
	} else if reply == "FAILED" {
		fmt.Println("Does not have the stock")
	}

	fmt.Println("From Channel status: ", reply)

	size, _, _ := connection.ReadFromUDP(buffer)
	timeGot := buffer[0:size]
	json.Unmarshal(timeGot, &tempTime)
	for system := 0; system < 3; system++ {
		if localTime.VectorTime[system] < tempTime.VectorTime[system] {
			localTime.VectorTime[system] = tempTime.VectorTime[system]
		}
	}

	localTime.VectorTime[localTime.ProcessID] += 1
}

func initial() {

	setUp, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49200")
	connection, err := net.DialUDP("udp4", nil, setUp)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection.Close()

	pId := "1" //sending the processID to the server
	pIdSend := []byte(pId + "\n")
	_, err = connection.Write(pIdSend)

	data := []byte("ALL" + "\n")
	_, err = connection.Write(data)
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		buffer := make([]byte, 1024)
		n, _, err := connection.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}
		dataFrom := string(buffer[0:n])
		dataFrom = strings.TrimSpace(string(dataFrom))
		if dataFrom == "EOF" {
			break
		}
		fmt.Println(dataFrom)
	}
}

func connectServer(ledgerKeeping map[[3]int]*ledger) {

	var in *bufio.Reader
	var price float64
	var available, order int
	in = bufio.NewReader(os.Stdin)

	setUp, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49200")
	connection, err := net.DialUDP("udp4", nil, setUp)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer connection.Close()

	pId := "1" //sending the processID to the server
	pIdSend := []byte(pId + "\n")
	_, err = connection.Write(pIdSend)

	fmt.Print("\nEnter the company's name: ")
	cName, _ := in.ReadString('\n')
	data := []byte(cName + "\n")
	_, err = connection.Write(data) //sending the name of the company wants
	if err != nil {
		fmt.Println(err)
		return
	}

	buffer := make([]byte, 1024)
	n, _, err := connection.ReadFromUDP(buffer) //getting the response from the server for the request (will be INVALID if company doesn't exist else it returns the price)
	if err != nil {
		fmt.Println(err)
		return
	}
	dataFrom := string(buffer[0:n])
	dataFrom = strings.TrimSpace(string(dataFrom))
	if dataFrom == "INVALID" {
		fmt.Println("This company doesn't exist")
		return
	} else {
		available, err = strconv.Atoi(dataFrom) //storing the no of stocks available
		if err != nil {
			fmt.Println(err)
		}

		number, _, err := connection.ReadFromUDP(buffer) //getting the price of the stock
		if err != nil {
			fmt.Println(err)
			return
		}
		dataPrice := string(buffer[0:number])
		dataPrice = strings.TrimSpace(string(dataPrice))
		price, err = strconv.ParseFloat(dataPrice, 64) //storing the price of the stock
		if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Printf(cName+"Offering %d", available)
	fmt.Printf(" stocks at %f", price)
	fmt.Print("\nEnter the number of stocks you would like to buy: ")
	fmt.Scanln(&order)
	response := strconv.Itoa(order)
	if err != nil {
		fmt.Println(err)
	}
	reply := []byte(response + "\n")
	_, err = connection.Write(reply)
	if err != nil {
		fmt.Println(err)
		return
	}
	no, _, err := connection.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println(err)
		return
	}
	dataGot := string(buffer[0:no])
	dataGot = strings.TrimSpace(string(dataGot))
	if strings.TrimSpace(string(dataGot)) == "SUCCESS" {
		fmt.Println("No of "+cName+" stocks bought: ", order)
		income = income - price*float64(order)
		ledgerKeeping[localTime.VectorTime] = NewLedger("1", cName, int64(order), price)
		offerings[cName] = New(cName, price, int64(order))

	} else if strings.TrimSpace(string(dataGot)) == "FAILED" {
		fmt.Println("Unable to buy try again later")
	}

}

func main() {

	ledgerKeeping = make(map[[3]int]*ledger)
	localTime = Clock{
		ProcessID:  1,
		VectorTime: [3]int{0, 0, 0},
	}
	offerings = make(map[string]*stock)
	go syncTimeReceiver()
	go receiveMarker()
	income = 0
	var choice, system int
	allStates = make(map[int]*State)
	stateID = 0
	Device0IP := "127.0.0.1:49201"
	Device2IP := "127.0.0.1:49203"

	channel0To1 = false
	channel2To1 = false

	for {
		stateRecorded = false

		fmt.Println("Vector time: ", localTime.VectorTime)
		fmt.Println("Enter the operation that you would like to perform:\n1.View all from central server\n2.Buy from central server\n3.Bid to other machines\n4.View income\n5.View Ledger\n6.View account\n7.View the transition states\n8.Shutdown")
		fmt.Scanln(&choice)
		localTime.VectorTime[localTime.ProcessID] += 1

		if choice == 1 {
			initial()
		} else if choice == 2 {
			connectServer(ledgerKeeping)
		} else if choice == 3 {
			fmt.Println("Enter the system you want to connect with (0 or 2): ")
			fmt.Scanln(&system)
			if system == 0 {
				syncTimeSender(Device0IP)
				sendRequest(Device0IP)
				sendMarker()
				time.Sleep(5 * time.Second)
			} else if system == 2 {
				syncTimeSender(Device2IP)
				sendRequest(Device2IP)
				sendMarker()
				time.Sleep(5 * time.Second)
			}
		} else if choice == 4 {
			fmt.Println("Income: ", income)

		} else if choice == 5 {
			localTime.VectorTime[localTime.ProcessID] += 1
			for keys := range ledgerKeeping {
				fmt.Printf(ledgerKeeping[keys].processID+" bought %d", ledgerKeeping[keys].stocksBought)
				fmt.Printf(" stocks of " + ledgerKeeping[keys].companyName)
				fmt.Printf(" for")
				fmt.Printf(" %f", ledgerKeeping[keys].stockPrice)
				fmt.Printf(" at ")
				fmt.Println(keys)
			}

		} else if choice == 6 {
			for keys := range offerings {
				View(offerings[keys])
			}
		} else if choice == 7 {
			for keys := range allStates {
				fmt.Printf("State %d", keys)
				viewState(allStates[keys])
			}
		} else if choice == 8 {
			fmt.Println("Shutting down the system")
			break
		} else {
			fmt.Println("Invalid entry")
		}
	}
}
