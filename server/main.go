package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/jinzhu/copier"
	pb "github.com/redcranetech/grpcspec-example/gogrpcspec"
	"google.golang.org/grpc"
)

type todoServer struct {
	tasks []pb.Task
	mu    sync.Mutex
}

func (s *todoServer) GetSummary(ctx context.Context, employee *pb.Employee) (*pb.SpecificSummary, error) {
	log.Println("GetSummary called.")
	specificSummary := pb.SpecificSummary{
		Employee: employee,
		Summary:  &pb.Summary{},
	}
	for _, task := range s.tasks {
		if task.GetEmployee().GetName() == employee.GetName() {
			if task.GetStatus() == "TODO" {
				specificSummary.Summary.TodoTasks++
			} else if task.GetStatus() == "DOING" {
				specificSummary.Summary.DoingTasks++
			} else {
				specificSummary.Summary.DoneTasks++
			}
		}
	}
	return &specificSummary, nil
}

func (s *todoServer) AddTask(ctx context.Context, taskToAdd *pb.Task) (*pb.SpecificSummary, error) {
	log.Println("AddTask called.")
	s.tasks = append(s.tasks, *taskToAdd)
	specificSummary := pb.SpecificSummary{
		Employee: taskToAdd.GetEmployee(),
		Summary:  &pb.Summary{},
	}
	for _, task := range s.tasks {
		if task.GetEmployee().GetName() == taskToAdd.GetEmployee().GetName() {
			if task.GetStatus() == "TODO" {
				specificSummary.Summary.TodoTasks++
			} else if task.GetStatus() == "DOING" {
				specificSummary.Summary.DoingTasks++
			} else {
				specificSummary.Summary.DoneTasks++
			}
		}
	}
	return &specificSummary, nil
}

func (s *todoServer) AddTasks(stream pb.TaskManager_AddTasksServer) error {
	log.Println("AddTasks called.")
	for {
		taskToAdd, err := stream.Recv()
		if err == io.EOF {
			summary := pb.Summary{}
			for _, task := range s.tasks {
				if task.GetStatus() == "TODO" {
					summary.TodoTasks++
				} else if task.GetStatus() == "DOING" {
					summary.DoingTasks++
				} else {
					summary.DoneTasks++
				}
			}
			return stream.SendAndClose(&summary)
		}
		if err != nil {
			return err
		}
		s.tasks = append(s.tasks, *taskToAdd)
	}
	return nil
}

func (s *todoServer) GetTasks(employee *pb.Employee, stream pb.TaskManager_GetTasksServer) error {
	log.Println("GetTasks called.")
	for _, task := range s.tasks {
		if task.GetEmployee().GetName() == employee.GetName() {
			if err := stream.Send(&task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *todoServer) ChangeToDone(stream pb.TaskManager_ChangeToDoneServer) error {
	log.Println("ChangeToDone called.")
	for {
		task, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.mu.Lock()
		for idx, t := range s.tasks {
			if task.GetEmployee().GetName() == t.GetEmployee().GetName() &&
				task.GetName() == t.GetName() {
				s.tasks[idx].Status = "DONE"
			}
		}
		tasksToBeStream := []pb.Task{}
		copier.Copy(&tasksToBeStream, &s.tasks)
		s.mu.Unlock()

		for _, t := range tasksToBeStream {
			if task.GetEmployee().GetName() == t.GetEmployee().GetName() &&
				t.GetStatus() != "DONE" {
				if err := stream.Send(&t); err != nil {
					return err
				}
			}

		}
	}
	return nil
}

func generateMockData() []pb.Task {
	employee1 := pb.Employee{
		Name: "Employee1",
	}
	employee2 := pb.Employee{
		Name: "Employee2",
	}
	tasks := []pb.Task{
		pb.Task{Employee: &employee1, Name: "Task1", Status: "TODO"},
		pb.Task{Employee: &employee1, Name: "Task2", Status: "DOING"},
		pb.Task{Employee: &employee1, Name: "Task3", Status: "DONE"},
		pb.Task{Employee: &employee1, Name: "Task4", Status: "DONE"},

		pb.Task{Employee: &employee2, Name: "Task5", Status: "TODO"},
		pb.Task{Employee: &employee2, Name: "Task6", Status: "TODO"},
		pb.Task{Employee: &employee2, Name: "Task7", Status: "TODO"},
		pb.Task{Employee: &employee2, Name: "Task8", Status: "DOING"},
		pb.Task{Employee: &employee2, Name: "Task9", Status: "DONE"},
	}
	return tasks

}
func main() {
	port := 5555
	log.Printf("Server starting on port: %d", port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	todoServer := todoServer{
		tasks: generateMockData(),
	}
	pb.RegisterTaskManagerServer(s, &todoServer)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
