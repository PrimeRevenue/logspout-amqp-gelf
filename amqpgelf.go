package amqpgelf

import (
	"os"
	"log"
	"time"
	"github.com/gliderlabs/logspout/router"
    "github.com/streadway/amqp"
    "github.com/Graylog2/go-gelf/gelf"
    "encoding/json"
    "strings"
    "fmt"

    "github.com/Jeffail/gabs"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
	router.AdapterFactories.Register(NewAmqpAdapter, "amqpgelf")
}


type AmqpAdapter struct {
    route     *router.Route
    address   string
    exchange  string
    exchange_type string
    key       string
    user      string
    password  string
}

func NewAmqpAdapter(route *router.Route) (router.LogAdapter, error) {
    address := route.Address

	// get our config value from the environment
    key := getenv("AMQP_ROUTING_KEY", "docker")
    exchange := getenv("AMQP_EXCHANGE", "log-messages")
    exchange_type := getenv("AMQP_EXCHANGE_TYPE", "direct")
    user := getenv("AMQP_USER", "guest")
    password := getenv("AMQP_PASSWORD", "guest")
    
	return &AmqpAdapter{
		route:    route,
        address:  address,
		exchange:    exchange,
		exchange_type: exchange_type,
		key:     key,
		user:    user,
		password:   password,
	}, nil     
    
}
func (a *AmqpAdapter) Stream(logstream chan *router.Message) {

        // Open AMQP connection to the URI
	connection, err := amqp.Dial("amqp://" + a.user + ":" + a.password + "@" + a.address)
    if err != nil {
       log.Fatalf("connection.open: %s - " + a.address, err)
    }
    // close the connection when function finishes
    defer connection.Close()
    
    // Open Channel on the connection
    channel, err := connection.Channel()
    if err != nil {
        log.Fatalf("channel.open: %s", err)
    }
    // Typically AMQP implementations would channel.ExchangeDeclare(foo bar etc) 
    // but we are assuming the exchange is created on RabbitMQ already
    for message := range logstream {
        m := &LogspoutMessage{message}
        level := gelf.LOG_INFO
        		if m.Source == "stderr" {
			level = gelf.LOG_ERR
		}
		extra, err := m.getExtraFields()
		if err != nil {
			log.Println("Graylog:", err)
			continue
		}

        prepared, err := prepareMessage(gelf.Message{
			                         Version:  "1.1",
			                         Host:     m.Container.Name[1:],
			                         Short:    m.Message.Data,
			                         TimeUnix: float64(m.Message.Time.UnixNano()/int64(time.Millisecond)) / 1000.0,
			                         Level:    level,
			                         Extra: extra,
            })
        if err != nil {
            log.Println("PrepareMessage:", err)
            continue
        }

        msg := amqp.Publishing{
        	//Headers: amqp.Table{},
        	//ContentType: "text/plain",
        	//ContentEncoding: "UTF-8",
    		//    DeliveryMode: amqp.Transient,
        	DeliveryMode: amqp.Persistent,
        	Priority: 0,
        	Timestamp:    time.Now(),
        	Body:         []byte(prepared),
    	}
        // Note mandatory is set to false, so it will be dropped if there are no
    	// queues bound to the exchange.
    	err = channel.Publish(
                    a.exchange, // exchange
                    a.key, // routing key
                    false, // mandatory
                    false, //immediate
                    msg)
    	if err != nil {
        // Since publish is asynchronous this can happen if the network connection
        // is reset or if the server has run out of resources.
        	log.Fatalf("basic.publish: %v", err)
    	}
	}
	
}    
func getenv(envkey string, default_value string) (value string) {
	value = os.Getenv(envkey)
	if value == "" {
		value = default_value
	}
	return
}

type LogspoutMessage struct {
	*router.Message
}

func (m LogspoutMessage) getExtraFields() (map[string]interface{}, error) {

	extra := map[string]interface{}{
		"_container_id":   m.Container.ID,
		"_container_name": m.Container.Name[1:], // might be better to use strings.TrimLeft() to remove the first /
		"_image_id":       m.Container.Image,
		"_image_name":     m.Container.Config.Image,
		"_command":        strings.Join(m.Container.Config.Cmd[:], " "),
		"_created":        m.Container.Created,
	}
	for name, label := range m.Container.Config.Labels {
		if strings.ToLower(name[0:5]) == "gelf_" {
			extra[name[4:]] = label
		}
	}
	swarmnode := m.Container.Node
	if swarmnode != nil {
		extra["_swarm_node"] = swarmnode.Name
	}

//	rawExtra, err := json.Marshal(extra)
//	if err != nil {
//		return nil, err
//	}
	return extra, nil
}


// prepareMessage marshal the given message, add extra fields and append EOL symbols
func prepareMessage(m gelf.Message) ([]byte, error) {
	// Marshal the GELF message in order to get base JSON
	jsonMessage, err := json.Marshal(m)
	if err != nil {
		return []byte{}, err
	}

	// Parse JSON in order to dynamically edit it
	c, err := gabs.ParseJSON(jsonMessage)
	if err != nil {
		return []byte{}, err
	}

	// Loop on extra fields and inject them into JSON
	for key, value := range m.Extra {
		_, err = c.Set(value, fmt.Sprintf("_%s", key))
		if err != nil {
			return []byte{}, err
		}
	}

	// Append the \n\0 sequence to the given message in order to indicate
	// to graylog the end of the message
	data := append(c.Bytes(), '\n', byte(0))

	return data, nil
}
