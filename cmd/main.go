package main

import (
	"context"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/zde37/Hive/internal/config"
	"github.com/zde37/Hive/internal/ipfs"
)

func main() {
	config := config.Load(os.Getenv("RPC_ADDR"), os.Getenv("WEB_UI_ADDR"), os.Getenv("GATEWAY_ADDR"))

	rpc, err := ipfs.NewClient(config.RPC_ADDR)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := ipfs.NewClientImpl(rpc, config.RPC_ADDR)

	// path, cid, err := client.Add(ctx, "./nt.txt")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("%s, %v", path, cid)

	// dir, err := os.Getwd()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Fatal(client.Get(ctx, "bafkreidmvzpgip324a56z6dx6pxdew3ow6uzkcz5tbqckydl4ikjtp2eie", fmt.Sprintf("%s/fileNameWithExtension.txt", dir)))

	// dir, err := os.Getwd()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Fatal(client.Get(ctx, "bafybeiewtvujdcaiesl6wh37wtz2y5qxx4bbcoymrq6ye5vw6jbcemlpsy", fmt.Sprintf("%s/directoryName", dir)))

	// res, err := client.Ping(ctx, "12D3KooWC9MzQ2WJzPqyKX71nzwjrudV1uL9goHyAFJJHNhAsHLV")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("First => %+v", res[0])
	// log.Printf("Last => %+v", res[len(res)-1])
	// log.Printf("Last => %+v",strings.ReplaceAll(res[len(res)-1].Text, "Average latency: ", ""))
	// log.Printf("Last => %+v", res[len(res)-1].Text)
	// log.Printf("Last => %+v", res[len(res)-1].Time)
	// log.Printf("Last => %+v", res[len(res)-1].Success)

	// res, err := client.NodeID(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("%+v",res)
	// log.Println(res.AgentVersion)
	// log.Println(res.ID)
	// log.Println(res.Protocols)
	// log.Println(res.PublicKey)
	// log.Println(res.Addresses)

	// files, err := client.ListPins(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println(files)

	// err = client.UnPinObject(ctx, "/ipfs/bafkreidmvzpgip324a56z6dx6pxdew3ow6uzkcz5tbqckydl4ikjtp2eie")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = client.PinObject(ctx, "/ipfs/bafkreidmvzpgip324a56z6dx6pxdew3ow6uzkcz5tbqckydl4ikjtp2eie")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// files, err := client.ListDir(ctx, "/ipfs/files/QmcPTgnjd9voKsG3iMwPDGydhMWanGQ3m2sxkzH5aPmFbY")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println(files)

	// err = client.FileLsRequest(ctx, "/files")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	res, err := client.DisplayFileContent(ctx, "/ipfs/QmcYsdRXMuV4ACQnqoG2URPLwtyQNCmNFEc3kPMymxQYbf")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(res)

	// peers, err := client.GetConnectedPeers(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("%+v", peers)
}
