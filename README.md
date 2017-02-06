# AMQP / GELF Module for Logspout
This module allows Logspout to send Docker logs in the GELF format to RabbitMQ via AMQP.

This project is a combination of a collection of code from several projects

The Gelf implementation from - https://github.com/micahhausler/logspout-gelf

JSON Marshaling from - https://github.com/Devatoria/go-graylog

and the Logspout -> AMQP implementation from - https://github.com/nicocesar/logspout

## Build
To build, see https://github.com/gliderlabs/logspout/tree/master/custom 

you will need add the following code to `modules.go` 

```
_ "github.com/PrimeRevenue/logspout-amqp-gelf"
```

## Configuration

AMQP configuration is passed in via container environment keys.

| Parameter | Default | Environment key |
|-----------|---------|-----------------|
| AMQP Routing Key | docker | AMQP_ROUTING_KEY |
| AMQP Exchange | log-messages | AMQP_EXCHANGE |
| AMQP Exchange Type | direct | AMQP_EXCHANGE_TYPE |
| AMQP User | guest | AMQP_USER |
| AMQP Password | guest | AMQP_PASSWORD |


## License
MIT. See [License](LICENSE)
