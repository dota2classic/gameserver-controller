package redis

const (
	BasePort = 30500
	MaxPort  = 35000
	KeyName  = "gameserver_port_counter"
)

func AllocateGameServerPorts() (int, int, error) {
	// Allocate two ports atomically
	val, err := Client.IncrBy(ctx, KeyName, 2).Result()
	if err != nil {
		return 0, 0, err
	}

	totalRange := MaxPort - BasePort
	offset := int(val % int64(totalRange))

	port1 := BasePort + offset
	port2 := port1 + 1
	if port2 >= MaxPort {
		port2 = BasePort
	}

	return port1, port2, nil
}
