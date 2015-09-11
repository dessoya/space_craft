
package buildings

import (
	"github.com/gocql/gocql"
	"fmt"
	"sc/logger"
)

type Dispatcher struct {

	PoolSize	uint16
	Workers		[]*Worker

}

func NewDispatcher(poolSize uint16) (*Dispatcher) {

	dispatcher := &Dispatcher{ PoolSize: poolSize, Workers: make([]*Worker, poolSize) }

	for index, _ := range dispatcher.Workers {
		worker := NewWorker()
		dispatcher.Workers[index] = worker
	}



	return dispatcher
}

func (d *Dispatcher) getFreeWorker() (*Worker) {

	return d.Workers[0]
}

func (d *Dispatcher) Build(PlanetUUID *gocql.UUID, bt int, x int, y int) {

	logger.String("Dispatcher.Build")

	worker := d.getFreeWorker()
	m := WorkerMessage{ Type: MT_Build, PlanetUUID: PlanetUUID, Params: make(map[string]interface{}) }
	m.Params["type"] = bt
	m.Params["x"] = x
	m.Params["y"] = y
	worker.C <- m

}

const (
 	MT_Quit int = iota
 	MT_Build
)

type WorkerMessage struct {
	Type int
	Params map[string]interface{}
	PlanetUUID *gocql.UUID
}

type Worker struct {

	C	chan WorkerMessage

}

func NewWorker() (*Worker) {
	worker := &Worker{ C: make(chan WorkerMessage, 128) }

	go worker.Loop()

	return worker
}

func (w *Worker) Loop() {

	for {

		m := <- w.C

		logger.String(fmt.Sprintf("%+v", m))

		switch m.Type {			

		}


	}
}
