package game

func (g *Game) refreshUnits() {
	if g.PepeClient == nil || g.refreshInProgress {
		return
	}

	g.refreshInProgress = true
	g.startSpinnerOp()

	client := g.PepeClient

	go func(ch chan<- refreshResult) {
		data, err := FetchUnits(client)
		ch <- refreshResult{data: data, err: err}
	}(g.refreshResultCh)
}

func (g *Game) sendMQTT(topicName, payload string) {
	if g.PepeClient == nil || g.PepeClient.GetMQTTClient() == nil || g.mqttInProgress {
		return
	}

	g.MQTTStatus = "MQTT: sending..."

	g.mqttInProgress = true
	g.startSpinnerOp()

	client := g.PepeClient

	go func(ch chan<- mqttResult) {
		var err error
		if client != nil && client.GetMQTTClient() != nil {
			err = client.GetMQTTClient().Publish(topicName, payload)
		}
		ch <- mqttResult{err: err}
	}(g.mqttResultCh)
}
