package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/jawher/mow.cli"

	. "github.com/Financial-Times/upp-next-video-mapper/logger"
	"github.com/Financial-Times/upp-next-video-mapper/video"
)

func main() {
	app := cli.App("next-video-mapper", "Catch native video content transform into Content and send back to queue.")

	addresses := app.Strings(cli.StringsOpt{
		Name:   "queue-addresses",
		Desc:   "Addresses to connect to the queue (hostnames).",
		EnvVar: "Q_ADDR",
	})

	group := app.String(cli.StringOpt{
		Name:   "group",
		Desc:   "Group used to read the messages from the queue.",
		EnvVar: "Q_GROUP",
	})

	readTopic := app.String(cli.StringOpt{
		Name:   "read-topic",
		Desc:   "The topic to read the meassages from.",
		EnvVar: "Q_READ_TOPIC",
	})

	readQueue := app.String(cli.StringOpt{
		Name:   "read-queue",
		Desc:   "The queue to read the meassages from.",
		EnvVar: "Q_READ_QUEUE",
	})

	writeTopic := app.String(cli.StringOpt{
		Name:   "write-topic",
		Desc:   "The topic to write the meassages to.",
		EnvVar: "Q_WRITE_TOPIC",
	})

	writeQueue := app.String(cli.StringOpt{
		Name:   "write-queue",
		Desc:   "The queue to write the meassages to.",
		EnvVar: "Q_WRITE_QUEUE",
	})

	authorization := app.String(cli.StringOpt{
		Name:   "authorization",
		Desc:   "Authorization key to access the queue.",
		EnvVar: "Q_AUTHORIZATION",
	})

	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "application port",
		EnvVar: "PORT",
	})

	app.Action = func() {
		InitLogs(os.Stdout, os.Stdout, os.Stderr)

		if len(*addresses) == 0 {
			ErrorLogger.Println("No queue address provided. Quitting...")
			cli.Exit(1)
		}

		consumerConfig := consumer.QueueConfig{
			Addrs:                *addresses,
			Group:                *group,
			Topic:                *readTopic,
			Queue:                *readQueue,
			ConcurrentProcessing: false,
			AutoCommitEnable:     true,
			AuthorizationKey:     *authorization,
		}

		producerConfig := producer.MessageProducerConfig{
			Addr:          (*addresses)[0],
			Topic:         *writeTopic,
			Queue:         *writeQueue,
			Authorization: *authorization,
		}

		handler := video.NewVideoMapperHandler(producerConfig)
		messageConsumer := consumer.NewConsumer(consumerConfig, handler.OnMessage, &http.Client{})
		InfoLogger.Println(prettyPrintConfig(consumerConfig, producerConfig))

		hc := video.NewHealthCheck(&producerConfig, &consumerConfig)

		go handler.Listen(hc, *port)
		consumeUntilSigterm(messageConsumer)
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func consumeUntilSigterm(messageConsumer consumer.MessageConsumer) {
	InfoLogger.Printf("Starting queue consumer: %#v", messageConsumer)
	var consumerWaitGroup sync.WaitGroup
	consumerWaitGroup.Add(1)

	go func() {
		messageConsumer.Start()
		consumerWaitGroup.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	messageConsumer.Stop()
	consumerWaitGroup.Wait()
}

func prettyPrintConfig(c consumer.QueueConfig, p producer.MessageProducerConfig) string {
	return fmt.Sprintf("Config: [\n\t%s\n\t%s\n]", prettyPrintConsumerConfig(c), prettyPrintProducerConfig(p))
}

func prettyPrintConsumerConfig(c consumer.QueueConfig) string {
	return fmt.Sprintf("consumerConfig: [\n\t\taddr: [%v]\n\t\tgroup: [%v]\n\t\ttopic: [%v]\n\t\treadQueueHeader: [%v]\n\t]", c.Addrs, c.Group, c.Topic, c.Queue)
}

func prettyPrintProducerConfig(p producer.MessageProducerConfig) string {
	return fmt.Sprintf("producerConfig: [\n\t\taddr: [%v]\n\t\ttopic: [%v]\n\t\twriteQueueHeader: [%v]\n\t]", p.Addr, p.Topic, p.Queue)
}
