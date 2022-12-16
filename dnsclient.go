package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	dns_server = "8.8.8.8:53"
)

func parseString(str string) []byte {
	pattern := make([]byte,len(str)+1)
	pattern[0] = '0'
	count := 0
	j := 1
	fillIndex := 0;
	for i:=0;i<len(str);i++ {
		if str[i] == '.' {
			pattern[fillIndex] = byte(count);
			pattern[j] = '0';
			count = 0;
			fillIndex = j;
		} else {
			pattern[j] = byte(str[i]);
			count += 1;
		}
		j = j + 1;
	}

	pattern[len(pattern)-1] = 0;
	//pattern = pattern[:cap(pattern)]

	return pattern;
}

func main() {


	if len(os.Args)<2 || len(os.Args)>2 {
		fmt.Print("Invalid Arguments")
		return;
	}

	args := os.Args[1:2]


	var website string
	if !strings.Contains(args[0],"www") {
		website = "www."+ args[0]
	} else {
		website = args[0]
	}

	//website := "www.mystica.me";
	website += "."

	payload := parseString(website)

	header := []byte{
		// transcation id : can be random
		0x12, 0x22,
		// flags
		// {QR: 0,OPCODE: "0000",AS:0,TRUNCATED: 0,RD: 1,RA: 0,Z: "000",RCODE: "0000"}
		// 000000010000 = 01 00
		0x01, 0x00,
		// Questions (Only one Question)
		0x00, 0x01,
		// Answer RRs (set by server)
		0x00, 0x00,
		// Authority RRs
		0x00, 0x00,
		// Additional RRs
		0x00, 0x00,
	}

	additional := []byte {
		// domainname, type A (Authoritative Server), class IN (Internet)
		// 0x03, 'w','w','w',
		// 0x07, 'm','y','s','t','i','c','a',
		// 0x02, 'm','e',
		// 0x00,

		0x00,0x01, // type as A
		0x00,0x01, // internet
	}

	query := append(payload,additional... )

	packet := append(header,query...)

	conn,err := net.Dial("udp",dns_server)

	if err!= nil {
		fmt.Println(err)
	}

	defer conn.Close()

	// make a request to dns server
	bytes_sent, err := conn.Write(packet)
	if err!= nil {
		fmt.Print(err)
	}

	fmt.Println("bytes transmitted",bytes_sent);

	// listen for response from the dns server!
	buffer := make([]byte,512)
	bytes_recv,err := conn.Read(buffer)
	if err!= nil {
		fmt.Print(err)
	}


	fmt.Print("\n ------------------Response!!!------------------------ \n\n")

	fmt.Println("bytes received",bytes_recv)

	// first twelve bytes are header
//	header_size := 12

	fmt.Printf("\nTransaction id: %d\n",binary.BigEndian.Uint16(buffer[0:2]))
	// fmt.Printf("Flags %d\n",binary.BigEndian.Uint16(buffer[2:4]))

	qn := binary.BigEndian.Uint16(buffer[4:6])
	fmt.Printf("Questions %d\n",qn)

	ans := binary.BigEndian.Uint16(buffer[6:8])
	fmt.Printf("Answers: %d\n",ans)
	fmt.Printf("Authority RR: %d\n",binary.BigEndian.Uint16(buffer[8:10]))
	fmt.Printf("Additional RR: %d\n",binary.BigEndian.Uint16(buffer[10:12]))


	offsetStart := 12;
	// question parse
	for i:=uint16(0);i<qn;i++ {
		fmt.Print("\nDomain:",getDomainName(buffer,offsetStart))
		fmt.Print("\nType:",binary.BigEndian.Uint16(buffer[28:30])) // a
		fmt.Print("\nClass:",binary.BigEndian.Uint16(buffer[30:32])) // internet
	}

	// Answer Section!

	curIndex := 32;
	for i:=uint16(0);i<ans;i++ {
		pointer := byte(buffer[curIndex]);

		// to check if first two bits are set or not, do bitwise on the binary!
		// 11000000 (192)
		// 00000011
		//       11
		// --------
		// 00000011 (3)

		firstTwoBitsSet := pointer >> 6 | 3;

		if firstTwoBitsSet==3 {

			fmt.Printf("\n         !!!!!!!!!!!!!!!! %d/%d  !!!!!!!!!!!!!           \n",i+1,ans)

			// yes, then get the next byte (offset)
			curIndex = curIndex + 1;
			offsetAddr := buffer[curIndex];
			curIndex = curIndex + 1;

			fmt.Print("\nDomain Name:",getDomainName(buffer,int(offsetAddr)))
			fmt.Print("\nDomain Type:",binary.BigEndian.Uint16(buffer[curIndex:curIndex+2]))
			curIndex = curIndex + 2;
		  fmt.Print("\nClass:",binary.BigEndian.Uint16(buffer[curIndex:curIndex + 2]))
			curIndex = curIndex + 2;

			var hex string
			for j:=0;j<4;j++ {
				hex += strconv.FormatInt(int64(buffer[curIndex]),16)
				curIndex = curIndex + 1;
			}

			decimal, _ := strconv.ParseInt(hex, 16, 32)

			fmt.Print("\nTTL: ",decimal)

			dataLength := binary.BigEndian.Uint16(buffer[curIndex:curIndex+2])
			curIndex = curIndex + 2;
			fmt.Print("\nIP: ")

			for k:=0;k<int(dataLength);k++ {
				fmt.Print(buffer[curIndex]);
				if k+1 <= int(dataLength)-1 {
					fmt.Print(".")
				}
				curIndex = curIndex + 1;
			}


		} else {
			fmt.Println("/end")
		}

	}

}

func getDomainName(buffer []byte,offsetStart int) string{

	a := buffer[offsetStart]
	currIndex := offsetStart + 1;
	var sb strings.Builder
	for ;buffer[currIndex]!=0; {
		if(a!=0) {
		sb.WriteString(string(buffer[currIndex]));
		currIndex = currIndex + 1;
		a = a - 1;

		if(a == 0) {
			if(buffer[currIndex+1]!=0) { sb.WriteString(".") }
			a = byte(buffer[currIndex]);
			currIndex = currIndex + 1;
		}
	}
	}
	return sb.String()
}
