package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type RunBody struct {
	Code string
}

func DockerRun(file string) string {
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

	err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		panic(err)
	}

	dockerStdOut := new(strings.Builder)
	dockerStdErr := new(strings.Builder)

	// return buf.String()
	_, err = stdcopy.StdCopy(dockerStdOut, dockerStdErr, out)
	if dockerStdErr.String() != "" {
		fmt.Println(dockerStdErr.String())
		return dockerStdErr.String()
	}

	return dockerStdOut.String()
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello!\n"))
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// In the future we could report back on the status of our DB, or our cache
	// (e.g. Redis) by performing a simple PING, and include them in the response.
	io.WriteString(w, `{"alive": true}`)
}

func RunHandler(w http.ResponseWriter, r *http.Request) {
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

	// write file
	err = os.WriteFile("test.py", []byte(jsonbody.Code), 0666)
	if err != nil {
		log.Fatal(err)
	}

	output := DockerRun("test.py")
	trimmedOutput := bytes.Trim([]byte(output), "\u0000")

	// w.Write([]byte("Testing!\n"))
	response := map[string]string{"message": string(trimmedOutput)}
	w.Header().Set("Content-Type", "application/json") // this
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	// get info from parameter
	vars := mux.Vars(r)
	category := vars["category"]
	filename := vars["id"]

	if len(vars["id"]) == 1 {
		filename = "0" + filename
	}

	filename = "static/" + category + "/" + filename

	if category == "info" {
		filename += ".md"
	} else if category == "code" {
		filename += ".py"
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file: %s, error msg:%s", filename, err)
		// panic(err)
		response := map[string]string{"message": string(err.Error())}

		w.Header().Set("Content-Type", "application/json") // this
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)

		return
	}

	text, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	// create response
	// text := "# hello world!\n## This is a markdown sample\nthis is a sample [link](https://www.google.com)\n"
	response := map[string]string{"message": string(text)}

	w.Header().Set("Content-Type", "application/json") // this
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI + " " + r.Method)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Gorilla Mux
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", TestHandler)
	r.HandleFunc("/health", HealthCheckHandler)
	r.HandleFunc("/run", RunHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/data/{category}/{id:[0-9]+}/", InfoHandler).Methods("GET", "OPTIONS")
	r.Use(loggingMiddleware)

	// cors
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"content-type"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
	)
	r.Use(cors)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
