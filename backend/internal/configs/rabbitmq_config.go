package configs

import "os"

type RabbitMQConfig struct {
	User     string
	Password string
	VHost    string
	Host     string
	Port     int
}

func loadRabbitMQConfig() (RabbitMQConfig, error) {
	port, err := portEnv("RABBITMQ_PORT", 0)
	if err != nil {
		return RabbitMQConfig{}, err
	}
	return RabbitMQConfig{
		User:     os.Getenv("RABBITMQ_DEFAULT_USER"),
		Password: os.Getenv("RABBITMQ_DEFAULT_PASS"),
		VHost:    os.Getenv("RABBITMQ_DEFAULT_VHOST"),
		Host:     os.Getenv("RABBITMQ_HOST"),
		Port:     port,
	}, nil
}
