package main

import (
	pb "chat_service/grpc"
	"flag"
	"io"
	"log"
	"math"
	"net"
	"strconv"

	"google.golang.org/grpc"
)

// Global variable
var server_lamport_time = 0
var port = flag.String("port", "", "server port")

// Keeps track of the server address, the port and connected clients
type Server struct {
	pb.UnimplementedChitChatServer
	address           string
	port              string
	connected_clients []Client
}

// Keeps track of the stream affiliated with a client identified by a username
type Client struct {
	stream    pb.ChitChat_ChatServer
	user_name string
}

func main() {
	flag.Parse()

	server := &Server{
		address: GetOutboundIP(),
		port:    *port,
	}

	go RunServer(server)

	for {

	}
}

// Creates, opens port and registers a server
func RunServer(server *Server) {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", server.address+":"+server.port)

	if err != nil {
		log.Fatalf("Could not create server %v", err)
	}

	IncrementAndPrintLamportTimestamp("Server listening")
	log.Printf("Server listening at: %v \n", listener.Addr().String())

	pb.RegisterChitChatServer(grpcServer, server)

	if serverError := grpcServer.Serve(listener); serverError != nil {
		log.Fatal("Could not serve listener")
	}
}

// Enable the service by listening to messages
func (server *Server) Chat(stream pb.ChitChat_ChatServer) error {
	err := RecieveMessages(stream, server)
	return err
}

// Recieves messages via the stream
func RecieveMessages(stream pb.ChitChat_ChatServer, server *Server) error {
	for {
		message, err := stream.Recv()

		SetAndPrintLamportTimestamp("Validate recieved timestamp", ValidateLamportTimestamp(int(message.GetLamportTimestamp()), server_lamport_time)+1)
		recieved_at_timestamp := server_lamport_time

		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Fatalf("An error has occured at lamport timestamp %v: %v", server_lamport_time, err)
		}

		HandleRecievedMessage(message, stream, server, recieved_at_timestamp)
	}
}

// Handles a message, depending on the type of message
func HandleRecievedMessage(message *pb.ChitChatInformationContainer, stream pb.ChitChat_ChatServer, server *Server, recieved_timestamp int) {

	if message.GetConnectionRequest() != nil {
		ConnectNewClient(server, message.GetConnectionRequest(), recieved_timestamp, &stream)
	} else if message.GetDisconnectionRequest() != nil {
		DisconnectClient(server, message.GetDisconnectionRequest(), recieved_timestamp)
	} else {
		incommin_message := message.GetMessage()

		user_name := incommin_message.GetUserName()
		message := incommin_message.GetMessage()

		IncrementAndPrintLamportTimestamp("Broadcast recieved message")
		Broadcast(server, user_name, message)
		log.Printf("%v: %v", incommin_message.GetUserName(), incommin_message.GetMessage())
	}
}

// Connects a new client to the server and broadcasts this change
func ConnectNewClient(server *Server, message *pb.ConnectionRequest, recieved_timestamp int, stream *pb.ChitChat_ChatServer) {
	user_name := message.GetUserName()

	IncrementAndPrintLamportTimestamp("Add new client")
	AddClientToClients(stream, server, user_name)

	IncrementAndPrintLamportTimestamp("Broadcast new client")
	Broadcast(server, "Server", GetFormattedConnectionMessage(user_name, recieved_timestamp))

	log.Printf("%v has joined the chat", message.GetUserName())
}

// Disconnects a client from a server and broadcasts this change
func DisconnectClient(server *Server, message *pb.DisconnectionRequest, recieved_timestamp int) {
	user_name := message.GetUserName()

	IncrementAndPrintLamportTimestamp("Disconnect client")
	for index, client := range server.connected_clients {
		if client.user_name == user_name {
			server.connected_clients = Remove(server.connected_clients, index)
		}
	}

	IncrementAndPrintLamportTimestamp("Broadcast client left")
	Broadcast(server, "Server", GetFormattedDisconnetionMessage(user_name, recieved_timestamp))

	log.Printf("%v has left the chat", message.GetUserName())
}

// Broadcasts a message to all connected clients
func Broadcast(server *Server, user_name string, message string) {
	container := CreateChitChatMessageObject(user_name, message)

	for _, client := range server.connected_clients {
		if err := client.stream.Send(container); err != nil {
			log.Fatalf("Error occured at lamport timestamp %v: %v", err, server_lamport_time)
		}
	}
}

// Appends a servers list of clients
func AddClientToClients(stream *pb.ChitChat_ChatServer, server *Server, user_name string) {
	server.connected_clients = append(server.connected_clients, Client{
		stream:    *stream,
		user_name: user_name,
	})
}

// Creates a ChitChat message rapped in a message container.
func CreateChitChatMessageObject(user_name string, message string) *pb.ChitChatInformationContainer {
	outbound_message := &pb.ChitChatMessage{UserName: user_name, Message: message}
	message_container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(server_lamport_time),
		These:            &pb.ChitChatInformationContainer_Message{Message: outbound_message},
	}
	return message_container
}

// Formats and returns a disconnection request
func GetFormattedDisconnetionMessage(user_name string, recived_timestamp int) string {
	return "Participant " + user_name + " left Chitty-Chat at Lamport time " + strconv.Itoa(recived_timestamp)
}

// Formats and returns a connection request
func GetFormattedConnectionMessage(user_name string, recieved_timestamp int) string {
	return "Participant " + user_name + " joined Chitty-Chat at Lamport time " + strconv.Itoa(recieved_timestamp)
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

// Validates wheter the client of the sever Lampert timestamp is the correct one.
func ValidateLamportTimestamp(client_timestamp int, server_timestamp int) int {
	return int(math.Max(float64(client_timestamp), float64(server_timestamp)))
}

func IncrementAndPrintLamportTimestamp(action string) {
	server_lamport_time++
	log.Printf("%v has incremented Lampert timestamp to: %v", action, server_lamport_time)
}

func SetAndPrintLamportTimestamp(action string, new_value int) {
	server_lamport_time = new_value
	log.Printf("%v has set Lampert timestamp to: %v", action, server_lamport_time)
}
