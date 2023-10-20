package main

import (
	pb "chat_service/grpc"
	"flag"
	"io"
	"log"
	"math"
	"net"

	"google.golang.org/grpc"
)

var server_lampert_timestamp = 0

type Server struct {
	pb.UnimplementedChitChatServer
	address string
	port    string
	clients []Client
}

type Client struct {
	stream    pb.ChitChat_ChatServer
	user_name string
}

var port = flag.String("port", "", "server port")

func main() {
	flag.Parse()

	server := &Server{
		address: GetOutboundIP(),
		port:    *port,
	}

	go StartServer(server)

	for {

	}
}

func StartServer(server *Server) {
	grpcServer := grpc.NewServer()
	listen, err := net.Listen("tcp", server.address+":"+server.port)

	if err != nil {
		log.Fatalf("Could not create server %v", err)
	}

	IncrementAndPrintLampertTimestamp("Server listening")
	log.Printf("Server started at: %v \n", listen.Addr().String())

	pb.RegisterChitChatServer(grpcServer, server)

	if serverError := grpcServer.Serve(listen); serverError != nil {
		log.Fatal("Could not serve listener")
	}
}

func (server *Server) Chat(stream pb.ChitChat_ChatServer) error {
	for {
		message, err := stream.Recv()
		IncrementAndPrintLampertTimestamp("Message recieved")

		SetAndPrintLampertTimestamp("Validate timestamp", ValidateLampertTimeStamp(int(message.GetLamportTimestamp()), server_lampert_timestamp))

		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Fatalf("An error has occured at lampert timestamp %v: %v", err, server_lampert_timestamp)
		}

		if message.GetConnectionRequest() != nil {
			ConnectNewClient(server, message.GetConnectionRequest(), &stream)
		} else if message.GetDisconnectionRequest() != nil {
			DisconnectClient(server, message.GetDisconnectionRequest())
		} else {

			incommin_message := message.GetMessage()

			outbound_message := &pb.ChitChatMessage{UserName: incommin_message.GetUserName(), Message: incommin_message.GetMessage()}

			IncrementAndPrintLampertTimestamp("Broadcast recieved message")
			Broadcast(server, outbound_message)
			log.Printf("%v: %v", incommin_message.GetUserName(), incommin_message.GetMessage())
		}
	}
}

func ConnectNewClient(server *Server, message *pb.ConnectionRequest, stream *pb.ChitChat_ChatServer) {
	server.clients = append(server.clients, Client{
		stream:    *stream,
		user_name: message.GetUserName(),
	})

	connection_message := &pb.ChitChatMessage{
		UserName: "Server",
		Message:  message.GetUserName() + " has joined the chat!",
	}

	IncrementAndPrintLampertTimestamp("Broadcast new client")
	Broadcast(server, connection_message)

	log.Printf("%v has joined the chat", message.GetUserName())
}

func DisconnectClient(server *Server, message *pb.DisconnectionRequest) {
	for index, client := range server.clients {
		if client.user_name == message.GetUserName() {
			server.clients = Remove(server.clients, index)
		}
	}

	disconnection_message := &pb.ChitChatMessage{
		UserName: "Server",
		Message:  message.GetUserName() + " has left the chat!",
	}

	IncrementAndPrintLampertTimestamp("Broadcast client left")
	Broadcast(server, disconnection_message)

	log.Printf("%v has left the chat", message.GetUserName())
}

func Broadcast(server *Server, message *pb.ChitChatMessage) {
	container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(server_lampert_timestamp),
		These: &pb.ChitChatInformationContainer_Message{
			Message: message,
		},
	}

	for _, client := range server.clients {
		if err := client.stream.Send(container); err != nil {
			log.Fatalf("Error occured at lampert timestamp %v: %v", err, server_lampert_timestamp)
		}
	}
}

// This function was found here: https://stackoverflow.com/a/37382208
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

// This function was found here: https://stackoverflow.com/a/37335777
func Remove(s []Client, i int) []Client {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func ValidateLampertTimeStamp(client_timestamp int, server_timestamp int) int {
	return int(math.Max(float64(client_timestamp), float64(server_timestamp)))
}

func IncrementAndPrintLampertTimestamp(action string) {
	server_lampert_timestamp++
	log.Printf("%v has incremented Lampert timestamp to: %v", action, server_lampert_timestamp)
}

func SetAndPrintLampertTimestamp(action string, new_value int) {
	server_lampert_timestamp = new_value
	log.Printf("%v has set Lampert timestamp to: %v", action, server_lampert_timestamp)
}
