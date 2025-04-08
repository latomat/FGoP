package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"mygrpcservice/proto"
)

const (
	serverAddr = "localhost:50051"
)

func main() {
	listCmd := flag.Bool("list", false, "List all uploaded files")
	uploadCmd := flag.String("upload", "", "Upload a file (provide file path)")
	downloadCmd := flag.String("download", "", "Download a file (provide file name)")
	outputPath := flag.String("out", "", "Output path for downloaded file (optional)")

	flag.Parse()

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("could not connect to server: %v", err)
	}
	defer conn.Close()

	client := proto.NewFileServiceClient(conn)

	switch {
	case *listCmd:
		listFiles(client)
	case *uploadCmd != "":
		uploadFile(client, *uploadCmd)
	case *downloadCmd != "":
		downloadFile(client, *downloadCmd, *outputPath)
	default:
		fmt.Println("Usage:")
		flag.PrintDefaults()
	}
}

func listFiles(client proto.FileServiceClient) {
	resp, err := client.ListFiles(context.Background(), &proto.Empty{})
	if err != nil {
		log.Fatalf("could not list files: %v", err)
	}

	fmt.Println("Files:")
	for _, file := range resp.GetFiles() {
		fmt.Printf("Name: %s, Created: %s, Updated: %s\n", file.GetName(), file.GetCreated(), file.GetUpdated())
	}
}

func uploadFile(client proto.FileServiceClient, filePath string) {
	stream, err := client.Upload(context.Background())
	if err != nil {
		log.Fatalf("could not start upload: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("could not open file: %v", err)
	}
	defer file.Close()

	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("error reading file: %v", err)
		}
		if n == 0 {
			break
		}

		err = stream.Send(&proto.FileChunk{
			Content:    buf[:n],
			ChunkSize:  int32(n),
		})
		if err != nil {
			log.Fatalf("could not send chunk: %v", err)
		}
	}

	status, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("upload failed: %v", err)
	}

	fmt.Printf("Upload status: success=%v, message=%s\n", status.Success, status.Message)
}

func downloadFile(client proto.FileServiceClient, fileName, outputPath string) {
	if outputPath == "" {
		outputPath = "./" + fileName
	}

	stream, err := client.Download(context.Background(), &proto.FileRequest{FileName: fileName})
	if err != nil {
		log.Fatalf("could not start download: %v", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("could not create output file: %v", err)
	}
	defer file.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving chunk: %v", err)
		}

		_, err = file.Write(chunk.Content)
		if err != nil {
			log.Fatalf("error writing chunk: %v", err)
		}
	}

	fmt.Printf("Downloaded %s to %s\n", fileName, outputPath)
}
