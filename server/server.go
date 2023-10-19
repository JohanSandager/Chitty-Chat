package main

import (
	proto "chat_service/grpc"
	"flag"
	"io"
	"log"
	"net"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedChitChatServer
	address string
	port    string
	clients []Client
}

type Client struct {
	stream    proto.ChitChat_ChatServer
	user_name string
}

var port = flag.String("port", "", "server port")

func main() {
	flag.Parse()

	server := &Server{
		address: GetOutboundIP(),
		port:    *port,
	}

	go startServer(server)

	for {

	}
}

func startServer(server *Server) {
	grpcServer := grpc.NewServer()
	listen, err := net.Listen("tcp", server.address+":"+server.port)

	if err != nil {
		log.Fatalf("Could not create server %v", err)
	}

	log.Printf("Server started at: %v \n", listen.Addr().String())

	proto.RegisterChitChatServer(grpcServer, server)
	serverError := grpcServer.Serve(listen)
	if serverError != nil {
		log.Fatal("Could not serve listener")
	}
}

func (server *Server) Chat(stream proto.ChitChat_ChatServer) error {
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Printf("An error has occured: %v", err)
		}
		if message.GetConnectionRequest() != nil {
			connectNewClient(server, message.GetConnectionRequest(), &stream)
		} else if message.GetDisconnectionRequest() != nil {
			disconnectClient(server, message.GetDisconnectionRequest())
		} else {

			incommin_message := message.GetMessage()

			outbound_message := &proto.ChitChatMessage{UserName: incommin_message.GetUserName(), Message: incommin_message.GetMessage()}

			broadcast(server, outbound_message)
			log.Printf("%v: %v", incommin_message.GetUserName(), incommin_message.GetMessage())
		}
	}
}

func connectNewClient(server *Server, message *proto.ConnectionRequest, stream *proto.ChitChat_ChatServer) {
	server.clients = append(server.clients, Client{
		stream:    *stream,
		user_name: message.GetUserName(),
	})

	connection_message := &proto.ChitChatMessage{
		UserName: "Server",
		Message:  message.GetUserName() + " has joined the chat!",
	}

	broadcast(server, connection_message)

	log.Printf("%v has joined the chat", message.GetUserName())
}

func disconnectClient(server *Server, message *proto.DisconnectionRequest) {
	for index, client := range server.clients {
		if client.user_name == message.GetUserName() {
			server.clients = remove(server.clients, index)
		}
	}

	disconnection_message := &proto.ChitChatMessage{
		UserName: "Server",
		Message:  message.GetUserName() + " has left the chat!",
	}

	broadcast(server, disconnection_message)

	log.Printf("%v has left the chat", message.GetUserName())
}

func broadcast(server *Server, message *proto.ChitChatMessage) {

	container := &proto.ChitChatInformationContainer{
		These: &proto.ChitChatInformationContainer_Message{
			Message: message,
		},
	}

	for _, client := range server.clients {
		if err := client.stream.Send(container); err != nil {
			log.Fatalf("Error occured: %v", err)
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
func remove(s []Client, i int) []Client {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
