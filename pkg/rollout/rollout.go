package rollout

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("rollout")
)

// Rollout is abstract class to secure that every node updates it's state one by one
type Rollout struct {
	leadershipElector     *leaderelection.LeaderElector
	startedLeadingChannel chan interface{}
}

// NewRollout creates new Rollou
func NewRollout(config *rest.Config, scheme *runtime.Scheme) (*Rollout, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to init clientSet: %v", err)
	}

	recorderProvider, err := NewProvider(clientSet, scheme, log.WithName("record_provider"))
	if err != nil {
		return nil, fmt.Errorf("fail to create new record provider: %v", err)
	}

	id := uuid.New().String()
	lock, err := resourcelock.New(resourcelock.ConfigMapsResourceLock, "nmstate", "leader-lock", clientSet.CoreV1(), clientSet.CoordinationV1(), resourcelock.ResourceLockConfig{
		Identity:      id,
		EventRecorder: recorderProvider.GetEventRecorderFor(id),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create resource lock: %v", err)
	}

	startedLeadingChannel := make(chan interface{})

	// create new leader elector
	leaderElector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 3 * time.Second,
		RenewDeadline: 2 * time.Second,
		RetryPeriod:   1 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				log.Info("Node is locked")
				startedLeadingChannel <- true
			},
			OnStoppedLeading: func() {
				close(startedLeadingChannel)
				log.Info("Node is unlocked")
			},
		},
	})
	if err != nil {
		close(startedLeadingChannel)
		return nil, fmt.Errorf("Error while creating new leader: %v", err)
	}

	return &Rollout{leadershipElector: leaderElector, startedLeadingChannel: startedLeadingChannel}, nil
}

// Lock locks current node and waits for unlock by calling the returned CancelFunc
func (rmgr *Rollout) Lock() context.CancelFunc {
	// create context which can stop current leader
	ctx, cancel := context.WithCancel(context.Background())

	// start election
	go rmgr.leadershipElector.Run(ctx)
	log.Info("Wait until node is locked")
	for !rmgr.leadershipElector.IsLeader() {
		<-rmgr.startedLeadingChannel
	}

	return cancel
}
