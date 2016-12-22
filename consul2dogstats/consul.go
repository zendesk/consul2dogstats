package consul2dogstats

type consulLock interface {
	Lock(stopCh <-chan struct{}) (<-chan struct{}, error)
	Unlock() error
	Destroy() error
}
