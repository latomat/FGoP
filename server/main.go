package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"google.golang.org/grpc"
	"mygrpcservice/proto"
)

const (
	port = ":50051"
)

type server struct {
	proto.UnimplementedFileServiceServer
	mu               sync.Mutex
	uploadStatus     map[string]string
	filesDir         string
	listSem          chan struct{} // ограничение на ListFiles
	transferSem      chan struct{} // ограничение на Upload/Download
}

func NewServer(filesDir string) *server {
	return &server{
		filesDir:     filesDir,
		uploadStatus: make(map[string]string),
		listSem:      make(chan struct{}, 100), // до 100 одновременных list-запросов
		transferSem:  make(chan struct{}, 10),  // до 10 одновременных upload/download
	}
}

// ListFiles - метод для получения списка файлов
func (s *server) ListFiles(ctx context.Context, in *proto.Empty) (*proto.FileList, error) {
	select {
	case s.listSem <- struct{}{}:
		defer func() { <-s.listSem }()
	default:
		return nil, fmt.Errorf("too many concurrent list requests")
	}

	files := []*proto.FileMetadata{}
	err := filepath.Walk(s.filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, &proto.FileMetadata{
				Name:    info.Name(),
				Created: info.ModTime().Format("2006-01-02 15:04:05"),
				Updated: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	return &proto.FileList{Files: files}, nil
}

// Upload - метод для загрузки файла
func (s *server) Upload(stream proto.FileService_UploadServer) error {
	select {
	case s.transferSem <- struct{}{}:
		defer func() { <-s.transferSem }()
	default:
		return fmt.Errorf("too many concurrent upload/download requests")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var fileName string

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		if fileName == "" {
			fileName = "uploaded_file"
		}

		filePath := fmt.Sprintf("%s/%s", s.filesDir, fileName)
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = file.Write(chunk.Content)
		if err != nil {
			return err
		}
	}

	s.uploadStatus[fileName] = "success"
	return stream.SendAndClose(&proto.UploadStatus{
		Success: true,
		Message: "File uploaded successfully",
	})
}

// Download - метод для скачивания файла
func (s *server) Download(req *proto.FileRequest, stream proto.FileService_DownloadServer) error {
	select {
	case s.transferSem <- struct{}{}:
		defer func() { <-s.transferSem }()
	default:
		return fmt.Errorf("too many concurrent upload/download requests")
	}

	filePath := fmt.Sprintf("%s/%s", s.filesDir, req.FileName)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("failed to read file: %v", err)
		}
		if n == 0 {
			break
		}

		err = stream.Send(&proto.FileChunk{
			Content:   buf[:n],
			ChunkSize: int32(n),
		})
		if err != nil {
			return fmt.Errorf("failed to send chunk: %v", err)
		}
	}
	return nil
}

func main() {
	filesDir := "./uploaded_files"
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		log.Fatalf("failed to create files directory: %v", err)
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterFileServiceServer(s, NewServer(filesDir))

	log.Printf("Server is running at %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
