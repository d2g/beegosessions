package memory

import (
	"container/list"
	"github.com/d2g/beegosessions/providers"
	"sync"
	"time"
)

var mempder = &MemProvider{list: list.New(), sessions: make(map[string]*list.Element)}

type MemProvider struct {
	lock        sync.RWMutex             //用来锁
	sessions    map[string]*list.Element //用来存储在内存
	list        *list.List               //用来做gc
	maxlifetime int64
	savePath    string
}

func (pder *MemProvider) SessionInit(maxlifetime int64, savePath string) error {
	pder.maxlifetime = maxlifetime
	pder.savePath = savePath
	return nil
}

func (pder *MemProvider) SessionRead(sid string) (providers.SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[sid]; ok {
		go pder.SessionUpdate(sid)
		pder.lock.RUnlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()
		newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[sid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
	return nil, nil
}

func (pder *MemProvider) SessionExist(sid string) bool {
	pder.lock.RLock()
	defer pder.lock.RUnlock()
	if _, ok := pder.sessions[sid]; ok {
		return true
	} else {
		return false
	}
}

func (pder *MemProvider) SessionRegenerate(oldsid, sid string) (providers.SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[oldsid]; ok {
		go pder.SessionUpdate(oldsid)
		pder.lock.RUnlock()
		pder.lock.Lock()
		element.Value.(*MemSessionStore).sid = sid
		pder.sessions[sid] = element
		delete(pder.sessions, oldsid)
		pder.lock.Unlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()
		newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[sid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
	return nil, nil
}

func (pder *MemProvider) SessionDestroy(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
		return nil
	}
	return nil
}

func (pder *MemProvider) SessionGC() {
	pder.lock.RLock()
	for {
		element := pder.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*MemSessionStore).timeAccessed.Unix() + pder.maxlifetime) < time.Now().Unix() {
			pder.lock.RUnlock()
			pder.lock.Lock()
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*MemSessionStore).sid)
			pder.lock.Unlock()
			pder.lock.RLock()
		} else {
			break
		}
	}
	pder.lock.RUnlock()
}

func (pder *MemProvider) SessionAll() int {
	return pder.list.Len()
}

func (pder *MemProvider) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*MemSessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
		return nil
	}
	return nil
}

func init() {
	providers.Register("memory", mempder)
}
