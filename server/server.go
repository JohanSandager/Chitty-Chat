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
			log.Print(err)
			return nil
		}
		if err != nil {
			log.Printf("An error has occured: %v", err)
		}
		if message.GetConnectionRequest() != nil {
			connectNewClient(server, message.GetConnectionRequest(), &stream)
			connection_message := &proto.ChitChatMessage{
				UserName: "Server",
				Message:  message.GetConnectionRequest().UserName + " has joined the chat!",
			}
			broadcast(server, connection_message)
		} else {

			outbound_message := &proto.ChitChatMessage{LamportTimestamp: message.GetMessage().GetLamportTimestamp(), UserName: message.GetMessage().GetUserName(), Message: message.GetMessage().GetMessage()}

			broadcast(server, outbound_message)
		}
	}
}

func connectNewClient(server *Server, client *proto.ConnectionRequest, stream *proto.ChitChat_ChatServer) {
	server.clients = append(server.clients, Client{
		stream:    *stream,
		user_name: client.UserName,
	})
	log.Printf("%v has joined the chat", client.UserName)
}

func broadcast(server *Server, message *proto.ChitChatMessage) {
	for _, client := range server.clients {
		if err := client.stream.Send(message); err != nil {
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
