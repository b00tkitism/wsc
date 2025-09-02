package main

import (
	"fmt"
	"net"
	"time"
)

func udpRoundTrip(pc net.PacketConn, addr string, payload []byte) ([]byte, net.Addr, error) {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, nil, err
	}
	_ = pc.SetDeadline(time.Now().Add(3 * time.Second)) // read+write deadline
	if _, err = pc.WriteTo(payload, raddr); err != nil {
		return nil, nil, err
	}
	buf := make([]byte, 2048)
	n, from, err := pc.ReadFrom(buf)
	if err != nil {
		return nil, nil, err
	}
	return buf[:n], from, nil
}

func main() {
	// Use your own PacketConn here if you have one
	pc, err := net.ListenPacket("udp", ":0")
	if err != nil {
		panic(err)
	}
	defer pc.Close()

	// Echo endpoint
	resp, from, err := udpRoundTrip(pc, "34.230.40.69:40000", []byte("hello from Go"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("echo from %v: %q\n", from, string(resp))

	// Info endpoint (returns JSON)
	resp, from, err = udpRoundTrip(pc, "34.230.40.69:40001", []byte("Get some UDP info"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("info from %v: %s\n", from, resp)
}

// package main

// import (
// 	"fmt"
// 	"net"
// )

// func main() {
// 	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer conn.Close()

// 	addr, err := net.ResolveUDPAddr("udp4", "34.230.40.69:40001")
// 	if err != nil {
// 		panic(err)
// 	}

// 	conn.WriteTo([]byte("hi hwo are you?"), addr)

// 	pack := make([]byte, 2048)
// 	n, from, err := conn.ReadFrom(pack)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("udp is : ", n, " | ", from, " | ", pack[:n], " | ", string(pack[:n]))
// }
