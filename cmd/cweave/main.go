package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cloudWeave/client"
)

func getEnvOrDefault(envKey, fallback string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return fallback
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	endpoint := getEnvOrDefault("CLOUDWEAVE_ENDPOINT", "http://localhost:8080")
	apiKey := getEnvOrDefault("CLOUDWEAVE_API_KEY", "default-admin-key")
	passphrase := getEnvOrDefault("CLOUDWEAVE_ENCRYPT_PASSPHRASE", "")

	switch command {
	case "put":
		putCmd := flag.NewFlagSet("put", flag.ExitOnError)
		keyFlag := putCmd.String("key", "", "object key name in storage")
		epFlag := putCmd.String("endpoint", endpoint, "CloudWeave node endpoint")
		keyAuthFlag := putCmd.String("api-key", apiKey, "API key for authentication")
		passFlag := putCmd.String("passphrase", passphrase, "client-side encryption passphrase")
		_ = putCmd.Parse(os.Args[2:])

		if putCmd.NArg() < 1 {
			fmt.Println("Error: missing file path to upload")
			fmt.Println("Usage: cweave put <file> [--key <object-key>]")
			os.Exit(1)
		}

		filePath := putCmd.Arg(0)
		objectKey := *keyFlag
		if objectKey == "" {
			objectKey = filepath.Base(filePath)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", filePath, err)
			os.Exit(1)
		}

		cli, err := client.New(client.Config{
			Endpoints:            []string{*epFlag},
			APIKey:               *keyAuthFlag,
			EncryptionPassphrase: *passFlag,
		})
		if err != nil {
			fmt.Printf("Error initializing client: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		if err := cli.Put(ctx, objectKey, data); err != nil {
			fmt.Printf("Upload failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully uploaded '%s' to key '%s' (%d bytes)\n", filePath, objectKey, len(data))

	case "get":
		getCmd := flag.NewFlagSet("get", flag.ExitOnError)
		outFlag := getCmd.String("out", "", "destination file path to save output")
		epFlag := getCmd.String("endpoint", endpoint, "CloudWeave node endpoint")
		keyAuthFlag := getCmd.String("api-key", apiKey, "API key for authentication")
		passFlag := getCmd.String("passphrase", passphrase, "client-side encryption passphrase")
		versionIDFlag := getCmd.String("version", "", "specific historical version ID to fetch")
		_ = getCmd.Parse(os.Args[2:])

		if getCmd.NArg() < 1 {
			fmt.Println("Error: missing object key")
			fmt.Println("Usage: cweave get <key> [--out <destination-path>]")
			os.Exit(1)
		}

		objectKey := getCmd.Arg(0)
		destPath := *outFlag
		if destPath == "" {
			destPath = filepath.Base(objectKey)
		}

		cli, err := client.New(client.Config{
			Endpoints:            []string{*epFlag},
			APIKey:               *keyAuthFlag,
			EncryptionPassphrase: *passFlag,
		})
		if err != nil {
			fmt.Printf("Error initializing client: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		var reader io.ReadCloser
		var info *client.ObjectInfo

		if *versionIDFlag != "" {
			reader, info, err = cli.GetVersion(ctx, objectKey, *versionIDFlag)
		} else {
			reader, info, err = cli.Get(ctx, objectKey)
		}

		if err != nil {
			fmt.Printf("Download failed: %v\n", err)
			os.Exit(1)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			fmt.Printf("Error reading downloaded content: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			fmt.Printf("Error writing file %s: %v\n", destPath, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully downloaded key '%s' to '%s' (%d bytes, Content-Type: %s)\n", objectKey, destPath, info.Size, info.ContentType)

	case "rm", "delete":
		rmCmd := flag.NewFlagSet("rm", flag.ExitOnError)
		epFlag := rmCmd.String("endpoint", endpoint, "CloudWeave node endpoint")
		keyAuthFlag := rmCmd.String("api-key", apiKey, "API key for authentication")
		_ = rmCmd.Parse(os.Args[2:])

		if rmCmd.NArg() < 1 {
			fmt.Println("Error: missing object key to remove")
			fmt.Println("Usage: cweave rm <key>")
			os.Exit(1)
		}

		objectKey := rmCmd.Arg(0)
		cli, err := client.New(client.Config{
			Endpoints: []string{*epFlag},
			APIKey:    *keyAuthFlag,
		})
		if err != nil {
			fmt.Printf("Error initializing client: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		if err := cli.Delete(ctx, objectKey); err != nil {
			fmt.Printf("Delete failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully deleted key '%s'\n", objectKey)

	case "versions":
		vCmd := flag.NewFlagSet("versions", flag.ExitOnError)
		epFlag := vCmd.String("endpoint", endpoint, "CloudWeave node endpoint")
		keyAuthFlag := vCmd.String("api-key", apiKey, "API key for authentication")
		_ = vCmd.Parse(os.Args[2:])

		if vCmd.NArg() < 1 {
			fmt.Println("Error: missing object key")
			fmt.Println("Usage: cweave versions <key>")
			os.Exit(1)
		}

		objectKey := vCmd.Arg(0)
		cli, err := client.New(client.Config{
			Endpoints: []string{*epFlag},
			APIKey:    *keyAuthFlag,
		})
		if err != nil {
			fmt.Printf("Error initializing client: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		versions, err := cli.ListVersions(ctx, objectKey)
		if err != nil {
			fmt.Printf("Listing versions failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Versions for key '%s' (%d total):\n", objectKey, len(versions))
		for idx, v := range versions {
			tag := ""
			if idx == len(versions)-1 {
				tag = " (latest)"
			}
			fmt.Printf("  - VersionID: %s | Size: %d bytes%s\n", v.VersionID, v.Size, tag)
		}

	case "ls":
		lsCmd := flag.NewFlagSet("ls", flag.ExitOnError)
		epFlag := lsCmd.String("endpoint", endpoint, "CloudWeave node endpoint")
		keyAuthFlag := lsCmd.String("api-key", apiKey, "API key for authentication")
		_ = lsCmd.Parse(os.Args[2:])

		req, err := http.NewRequest(http.MethodGet, *epFlag+"/cluster/status", nil)
		if err != nil {
			fmt.Printf("Failed to create request: %v\n", err)
			os.Exit(1)
		}
		if *keyAuthFlag != "" {
			req.Header.Set("Authorization", "Bearer "+*keyAuthFlag)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Failed to query cluster status from %s: %v\n", *epFlag, err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var status map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&status)

		fmt.Printf("CloudWeave Cluster Status (%s):\n", *epFlag)
		if nodes, ok := status["active_nodes"].([]interface{}); ok {
			fmt.Printf("  Active Nodes (%d):\n", len(nodes))
			for _, n := range nodes {
				fmt.Printf("    - %v\n", n)
			}
		}
		fmt.Printf("  Total File Uploads: %v\n", status["file_uploads_total"])
		fmt.Printf("  Total File Downloads: %v\n", status["file_downloads_total"])
		fmt.Printf("  Repaired Chunks: %v\n", status["repaired_chunks_total"])

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CloudWeave Command Line Interface (cweave)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cweave put <file> [--key <key>] [--passphrase <pass>]    Upload a file to object storage")
	fmt.Println("  cweave get <key> [--out <destination>] [--version <id>]  Download an object key")
	fmt.Println("  cweave versions <key>                                    List historical versions of key")
	fmt.Println("  cweave rm <key>                                          Delete an object key")
	fmt.Println("  cweave ls                                                Show cluster status and active nodes")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CLOUDWEAVE_ENDPOINT            Default node address (http://localhost:8080)")
	fmt.Println("  CLOUDWEAVE_API_KEY             Default API key")
	fmt.Println("  CLOUDWEAVE_ENCRYPT_PASSPHRASE  Client-side AES encryption passphrase")
}
