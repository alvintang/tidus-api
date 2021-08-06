package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gorilla/mux"
)

type RunBody struct {
	Code string
}

func DockerRun(file string) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	// io.Copy(os.Stdout, reader)

	// get dir
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// fmt.Println(pwd)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      "python",
		Cmd:        []string{"python", file},
		Tty:        false,
		WorkingDir: "/usr/src/app",
	}, &container.HostConfig{
		// AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: pwd,
				Target: "/usr/src/app",
			},
		},
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

func RunHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Testing!\n"))

	// fmt.Printf("headers:\n%s\n", r.Header)

	// Parse body
	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("body:\n%s\n", string(body))
	// end parse

	decoder := json.NewDecoder(r.Body)
	var jsonbody RunBody
	err := decoder.Decode(&jsonbody)

	if err != nil {
		panic(err)
	}

	// fmt.Println("body:" + jsonbody.Code)

	// write file
	err = os.WriteFile("test.py", []byte(jsonbody.Code), 0666)
	if err != nil {
		log.Fatal(err)
	}

	DockerRun("test.py")

	// cmd := exec.Command("docker", "run", "-v \"$PWD\":/usr/src/app -w /usr/src/app python python hello.py")
	// // cmd := exec.Command("docker", "ps", "-a")
	// fmt.Printf("output:\n%s\n", cmd.String())
	// out, err := cmd.Output()
	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// }

	// fmt.Printf("output:\n%s\n", out)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Gorilla Mux
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/run", RunHandler).Methods("POST")
	r.Use(loggingMiddleware)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
