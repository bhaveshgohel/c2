package main

import (
	"log"

	e4 "teserakt/e4common"
)

func (s *C2) newClient(id, key []byte) error {

	logger := log.With(c2.logger, "protocol", "e4", "command", "newClient")

	err := s.insertIDKey(id, key)
	if err != nil {
		logger.Log("msg", "insertIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID)
	return nil
}

func (s *C2) removeClient(id []byte) error {

	logger := log.With(c2.logger, "protocol", "e4", "command", "removeClient")

	err := s.deleteIDKey(id)
	if err != nil {
		logger.Log("msg", "deleteIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID)
	return nil
}

func (s *C2) newTopicClient(id []byte, topic string) error {

	logger := log.With(c2.logger, "protocol", "e4", "command", "newTopicClient")

	key, err := s.getTopicKey(topic)
	if err != nil {
		logger.Log("msg", "getTopicKey failed", "error", err)
		return err
	}

	topichash := e4.HashTopic(topic)

	payload, err := s.CreateAndProtectForID(e4.SetTopicKey, topichash, key, id)
	if err != nil {
		logger.Log("msg", "CreateAndProtectForID failed", "error", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID, "topic", topic)
	return nil
}

func (s *C2) removeTopicClient(id []byte, topic string) error {

	logger := log.With(c2.logger, "protocol", "e4", "command", "removeTopicClient")

	topichash := e4.HashTopic(topic)

	payload, err := s.CreateAndProtectForID(e4.RemoveTopic, topichash, nil, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in removeTopicClient: ", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		log.Print("sendCommandToClient failed in removeTopicClient", err)
		return err
	}

	log.Printf("removed topic '%s' from client %s", topic, e4.PrettyID(id))
	return nil
}

func (s *C2) resetClient(id []byte) error {

	payload, err := s.CreateAndProtectForID(e4.ResetTopics, nil, nil, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in resetClient: ", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		log.Print("sendCommandToClient failed in resetClient: ", err)
		return err
	}

	log.Printf("reset client %s", e4.PrettyID(id))
	return nil
}

func (s *C2) newTopic(topic string) error {

	key := e4.RandomKey()

	err := s.insertTopicKey(topic, key)
	if err != nil {
		log.Print("insertTopicKey failed in newTopic: ", err)
		return err
	}
	log.Printf("added topic %s", topic)
	return nil
}

func (s *C2) removeTopic(topic string) error {

	err := s.deleteTopicKey(topic)
	if err != nil {
		log.Print("deleteTopic failed in removeTopic: ", err)
		return err
	}
	log.Printf("removed topic %s", topic)
	return nil
}

func (s *C2) sendMessage(topic, msg string) error {
	topickey, err := s.getTopicKey(topic)
	if err != nil {
		return err
	}
	payload, err := e4.Protect([]byte(msg), topickey)
	if err != nil {
		return err
	}
	err = s.publish(payload, topic, byte(0))
	if err != nil {
		log.Print("publish failed in sendMessage: ", err)
		return err
	}
	return nil
}

func (s *C2) newClientKey(id []byte) error {

	key := e4.RandomKey()

	// first send to the client, and only update locally afterwards
	payload, err := s.CreateAndProtectForID(e4.SetIDKey, nil, key, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in newClientKey: ", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		log.Print("sendCommandToClient failed in newClientKey: ", err)
		return err
	}

	err = s.insertIDKey(id, key)
	if err != nil {
		log.Print("insertIDKey failed in newClientKey: ", err)
		return err
	}
	log.Printf("updated key for client %s", e4.PrettyID(id))
	return nil
}
