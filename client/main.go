package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/redcranetech/grpcspec-example/gogrpcspec"
	"google.golang.org/grpc"
)

func unaryGetSummary(client pb.TaskManagerClient) {
	req := pb.Employee{Name: "Employee1"}
	resp, err := client.GetSummary(context.Background(), &req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("GetSummary response:\n%+v\n", resp)
}

func unaryAddTask(client pb.TaskManagerClient) {
	task := pb.Task{
		Employee: &pb.Employee{Name: "Employee1"},
		Name:     "New Task",
		Status:   "TODO",
	}
	req := task
	resp, err := client.AddTask(context.Background(), &req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("AddTask response: \n%+v\n", resp)
}

func clientStreamAddTask(client pb.TaskManagerClient) {
	tasks := []pb.Task{
		pb.Task{Employee: &pb.Employee{Name: "Employee1"}, Name: "NewTask1", Status: "TODO"},
		pb.Task{Employee: &pb.Employee{Name: "Employee1"}, Name: "NewTask2", Status: "TODO"},
		pb.Task{Employee: &pb.Employee{Name: "Employee1"}, Name: "NewTask3", Status: "TODO"},
	}
	stream, err := client.AddTasks(context.Background())
	if err != nil {
		log.Fatalf("%v.AddTasks(_) = _, %v", client, err)
	}
	for _, task := range tasks {
		if err := stream.Send(&task); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, task, err)
		}
	}
	resp, err := stream.CloseAndRecv()
	fmt.Printf("AddTasks response: \n%+v\n", resp)
}

func serverStreamGetTask(client pb.TaskManagerClient) {
	stream, err := client.GetTasks(context.Background(), &pb.Employee{Name: "Employee1"})
	if err != nil {
		log.Fatalf("%v.GetTasks(_) = _, %v", client, err)
	}
	fmt.Println("GetTasks responses:")
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.GetTasks(_) = _, %v", client, err)
		}
		fmt.Println(task)
	}
}

func biDirectionalStreamChangeToDone(client pb.TaskManagerClient) {
	fmt.Println("ChangeToDone")
	stream, err := client.ChangeToDone(context.Background())
	if err != nil {
		log.Fatalf("%v.ChangeToDone(_) = _, %v", client, err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			fmt.Printf("Remaining task: %+v\n", t)
		}
	}()
	doneTasks := []pb.Task{
		pb.Task{Employee: &pb.Employee{Name: "Employee1"}, Name: "Task1"},
		pb.Task{Employee: &pb.Employee{Name: "Employee1"}, Name: "Task2"},

		// pb.Task{Employee: &pb.Employee{Name: "Employee2"}, Name: "Task5"},
		// pb.Task{Employee: &pb.Employee{Name: "Employee2"}, Name: "Task6"},
		// pb.Task{Employee: &pb.Employee{Name: "Employee2"}, Name: "Task7"},
	}
	for _, t := range doneTasks {
		fmt.Printf("-->Stamping done for: %+v\n", t)
		if err := stream.Send(&t); err != nil {
			log.Fatalf("Failed to send a note: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
	stream.CloseSend()
	<-waitc
}

func main() {
	conn, err := grpc.Dial(":5555", grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to backend: %v\n", err)
		os.Exit(1)
	}
	client := pb.NewTaskManagerClient(conn)
	unaryGetSummary(client)
	fmt.Println("=========")
	unaryAddTask(client)
	fmt.Println("=========")
	clientStreamAddTask(client)
	fmt.Println("=========")
	serverStreamGetTask(client)
	fmt.Println("=========")
	biDirectionalStreamChangeToDone(client)
}
