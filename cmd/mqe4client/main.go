/*
 * Copyright (c) 2013 IBM Corp.
 *
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *    Seth Hoenig
 *    Allan Stockdill-Mander
 *    Mike Robertson
 *
 * With modifications by JP Aumasson <jp@teserakt.io>
 * Copyright (c) 2018 Teserakt AG
 *
 */

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	e4 "teserakt/e4go/pkg/e4client"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// variables set at build time
var gitCommit string
var buildDate string

// E4: hardcoded values for testing
const (
	E4IdAlias  = "testid"
	E4Pwd      = "testpwd"
	E4FilePath = "./client.e4p"
)

/*
 Options:
  [-help]                      Display help
  [-a pub|sub]                 Action pub (publish) or sub (subscribe)
  [-m <message>]               Payload to send
  [-n <number>]                Number of messages to send or receive
  [-q 0|1|2]                   Quality of Service
  [-clean]                     CleanSession (true if -clean is present)
  [-id <clientid>]             CliendID
  [-user <user>]               User
  [-password <password>]       Password
  [-broker <uri>]              Broker URI
  [-topic <topic>]             Topic
  [-store <path>]              Store Directory
*/

func main() {
	fmt.Println("    /---------------------------------/")
	fmt.Println("   /  E4: MQTT test client           /")
	fmt.Printf("  /  version %s-%s          /\n", buildDate, gitCommit[:4])
	fmt.Println(" /  Teserakt AG, 2018              /")
	fmt.Println("/---------------------------------/")
	fmt.Println("")

	topic := flag.String("topic", "", "The topic name to/from which to publish/subscribe")
	broker := flag.String("broker", "tcp://test.mosquitto.org:1883", "The broker URI. ex: tcp://10.10.1.1:1883")
	password := flag.String("password", "", "The password (optional)")
	user := flag.String("user", "", "The User (optional)")
	id := flag.String("id", "testid", "The ClientID (optional)")
	cleansess := flag.Bool("clean", false, "Set Clean Session (default false)")
	qos := flag.Int("qos", 0, "The Quality of Service 0,1,2 (default 0)")
	num := flag.Int("num", 1, "The number of messages to publish or subscribe (default 1)")
	payload := flag.String("message", "", "The message text to publish (default empty)")
	action := flag.String("action", "", "Action publish or subscribe (required)")
	store := flag.String("store", ":memory:", "The Store Directory (default use memory store)")
	flag.Parse()

	if *action != "pub" && *action != "sub" {
		fmt.Println("Invalid setting for -action, must be pub or sub")
		return
	}

	if *topic == "" {
		fmt.Println("Invalid setting for -topic, must not be empty")
		return
	}

	fmt.Printf("Sample Info:\n")
	fmt.Printf("\taction:    %s\n", *action)
	fmt.Printf("\tbroker:    %s\n", *broker)
	fmt.Printf("\tclientid:  %s\n", *id)
	fmt.Printf("\tuser:      %s\n", *user)
	fmt.Printf("\tpassword:  %s\n", *password)
	fmt.Printf("\ttopic:     %s\n", *topic)
	fmt.Printf("\tmessage:   %s\n", *payload)
	fmt.Printf("\tqos:       %d\n", *qos)
	fmt.Printf("\tcleansess: %v\n", *cleansess)
	fmt.Printf("\tnum:       %d\n", *num)
	fmt.Printf("\tstore:     %s\n", *store)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetClientID(*id)
	opts.SetUsername(*user)
	opts.SetPassword(*password)
	opts.SetCleanSession(*cleansess)
	if *store != ":memory:" {
		opts.SetStore(MQTT.NewFileStore(*store))
	}

	log.SetPrefix("mqe4client\t")

	// E4: creating a fresh client, rather than loading from disk
	e4Client := e4.NewClientPretty(E4IdAlias, E4Pwd, E4FilePath)
	fmt.Printf("E4 client created with id %s\n", hex.EncodeToString(e4Client.ID))

	if *action == "pub" {
		client := MQTT.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}
		fmt.Println("Sample Publisher Started")
		for i := 0; i < *num; i++ {
			// E4: if topic key available, encrypt payload
			protected, err := e4Client.Protect([]byte(*payload), *topic)
			if err == nil {
				log.Println("---- doing publish (E4-protected) ----")
				token := client.Publish(*topic, byte(*qos), false, protected)
				token.Wait()
			} else if err == e4.ErrTopicKeyNotFound {
				// E4: if topic key not found, publish unencrypted
				log.Println("---- doing publish (NOT E4-protected) ----")
				token := client.Publish(*topic, byte(*qos), false, *payload)
				token.Wait()
			} else {
				// another error occured, don't publish
				log.Printf("E4 error: %s", err)
			}
		}

		client.Disconnect(250)
		fmt.Println("Sample Publisher Disconnected")
	} else {
		receiveCount := 0
		choke := make(chan [2]string)

		opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
			choke <- [2]string{msg.Topic(), string(msg.Payload())}
		})

		client := MQTT.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		if token := client.Subscribe(*topic, byte(*qos), nil); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			os.Exit(1)
		}

		// E4/ subscribe to receiving topic
		if token := client.Subscribe(e4Client.ReceivingTopic, byte(2), nil); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			os.Exit(1)
		}

		for receiveCount < *num {
			incoming := <-choke
			// E4: if topic is E4/<id>, the process as a command
			if incoming[0] == e4Client.ReceivingTopic {
				cmd, err := e4Client.ProcessCommand([]byte(incoming[1]))
				if err != nil {
					log.Printf("E4 error in ProcessCommand: %s\n", err)
				} else {
					log.Printf("received command %s\n", cmd)
				}
			} else {
				// E4: attempt to decrypt
				message, err := e4Client.Unprotect([]byte(incoming[1]), incoming[0])
				if err == nil {
					log.Printf("received (E4-protected) on topic: %s: %s\n", incoming[0], message)
				} else if err == e4.ErrTopicKeyNotFound {
					log.Printf("received (NOT E4-protected) on topic: %s: %s\n", incoming[0], incoming[1])
				} else {
					log.Printf("E4 error: %s", err)
				}
			}
			receiveCount++
		}

		client.Disconnect(250)
		fmt.Println("Sample Subscriber Disconnected")
	}
}