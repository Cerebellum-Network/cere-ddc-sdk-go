package domain

import (
	"bytes"
	"fmt"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/crypto"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

const (
	uri = "/state"
)

type Redirects struct {
	Soft uint32
	Hard uint32
}

type Statistic struct {
	Sessions  uint32
	Redirects Redirects
	CacheSize uint32
	CPU       uint8
	RAM       uint8
	HD        uint8
	Uptime    int64
}

type ShortState struct {
	NodeID    uint32
	ClusterID uint32
	Url       string
	State     NodeState
	Location  string
	Size      uint8
	Updated   int64
}

type Check struct {
	SateCheck ShortState
	Signature StateSignature
}

type GossipState struct {
	LastState ShortState
	Signature StateSignature
	Statistic Statistic
	Checks    map[uint32]Check
}

type CDNState struct {
	ShortState     ShortState
	Signature      StateSignature
	Statistic      Statistic
	ClusterMap     map[uint32]*GossipState
	bucketContract bucket.DdcBucketContract
	mx             sync.RWMutex
	startTime      int64
	scheme         crypto.Scheme
	gossipChance   int
}

type Cfg struct {
	ClusterId           uint32
	NodeId              uint32
	Url                 string
	Location            string
	Size                uint8
	Scheme              crypto.Scheme
	DetailedState       *Statistic
	UpdateSCInterval    time.Duration
	UpdateP2PInterval   time.Duration
	UpdateStateInterval time.Duration
	GossipChance        int // 1/n chance, that node state will be collected via HTTP
}

func CreateCDNState(cdnContract bucket.DdcBucketContract, cfg Cfg) *CDNState {
	s := &CDNState{
		bucketContract: cdnContract,
		startTime:      time.Now().Unix(),
		scheme:         cfg.Scheme,
		ShortState: ShortState{
			NodeID:    cfg.NodeId,
			ClusterID: cfg.ClusterId,
			Url:       cfg.Url,
			State:     Grey,
			Location:  cfg.Location,
			Size:      cfg.Size,
			Updated:   time.Now().Unix(),
		},
		Signature: StateSignature{
			Scheme:        crypto.SchemeName(cfg.Scheme.Name()),
			PublicKey:     encodeHex(cfg.Scheme.PublicKey()),
			MultiHashType: 0,
			Signature:     []byte{},
		},
		Statistic:    Statistic{},
		ClusterMap:   make(map[uint32]*GossipState, 0),
		gossipChance: cfg.GossipChance,
	}
	if err := s.updateStateSignature(); err != nil {
		log.WithError(err).Warning("Failed to sign default simple state")
	}
	if err := s.updateClusterMapFromSmartContractState(); err != nil {
		log.WithError(err).Warning("Failed to update state topologies from SC")
	}
	if err := s.selfUpdate(); err != nil {
		log.WithError(err).Warning("Failed to update self state")
	}
	go func() {
		ticker := time.NewTicker(cfg.UpdateSCInterval)
		for range ticker.C {
			if err := s.updateClusterMapFromSmartContractState(); err != nil {
				log.WithError(err).Warning("Failed to update state topologies from SC")
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.UpdateP2PInterval)
		for range ticker.C {
			s.updateClusterMapFromGossips()
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.UpdateStateInterval)
		for range ticker.C {
			if err := s.selfUpdate(); err != nil {
				log.WithError(err).Warning("Failed to update self state")
			}
		}
	}()
	return s
}

func (ss *ShortState) cidMessage(multiHashType uint64, t uint64) (message []byte, shortStateCid string, err error) {
	stateStr := fmt.Sprintf("%d%d%s%d%s%d%d", ss.NodeID, ss.ClusterID, ss.Url, ss.State, ss.Location, ss.Size, ss.Updated)
	shortStateCid, err = cid.CreateBuilder(multiHashType).Build([]byte(stateStr))
	if err != nil {
		return nil, "", err
	}

	if t == 0 {
		return []byte(shortStateCid), shortStateCid, nil
	}

	timeText := formatTimestamp(t)
	msg := fmt.Sprintf("<Bytes>DDC store %s at %s</Bytes>", shortStateCid, timeText)

	return []byte(msg), shortStateCid, nil
}

func (s *CDNState) updateStateSignature() error {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.Signature.Timestamp = uint64(time.Now().Unix())
	stateMessage, _, err := s.ShortState.cidMessage(s.Signature.MultiHashType, s.Signature.Timestamp)
	if err != nil {
		return err
	}
	sig, err := s.scheme.Sign(stateMessage)
	if err != nil {
		return err
	}
	s.Signature.Signature = encodeHex(sig)
	return nil
}

func validateSignature(ss ShortState, sig StateSignature) (bool, error) {
	if sig.Signature == nil {
		return false, fmt.Errorf("signature is nil")
	}
	if sig.PublicKey == nil {
		return false, fmt.Errorf("public key is nil")
	}
	if sig.Scheme == "" {
		return false, fmt.Errorf("scheme is empty")
	}
	if sig.Timestamp == 0 {
		return false, fmt.Errorf("timestamp is zero")
	}
	stateMessage, _, err := ss.cidMessage(sig.MultiHashType, sig.Timestamp)
	if err != nil {
		return false, err
	}
	decodedSignature, err := sig.DecodedSignature()
	if err != nil {
		return false, err
	}
	decodedKey, err := sig.DecodedPublicKey()
	if err != nil {
		return false, err
	}
	return crypto.Verify(sig.Scheme, decodedKey, stateMessage, decodedSignature)
}

func (s *CDNState) selfUpdate() error {
	stats := s.Statistic
	// TODO: add more logic
	shortState := s.ShortState
	shortState.State = Green
	shortState.Updated = time.Now().Unix()

	s.mx.Lock()
	defer s.mx.Unlock()
	s.ShortState = shortState
	err := s.updateStateSignature()
	if err != nil {
		return err
	}
	s.Statistic = stats
	return nil
}

func (s *CDNState) updateClusterMapFromSmartContractState() error {
	rawClusterState, err := s.bucketContract.CDNClusterGet(s.ShortState.ClusterID)
	if err != nil {
		return err
	}
	rawNodesState := make(map[uint32]*bucket.CDNNodeStatus, len(rawClusterState.CDNCluster.CDNNodes))
	for _, id := range rawClusterState.CDNCluster.CDNNodes {
		rawNodesState[id], err = s.bucketContract.CDNNodeGet(id)
		if err != nil {
			return err
		}
	}
	// Check that current node exist on the smart contract
	if _, ok := rawNodesState[s.ShortState.NodeID]; !ok {
		return fmt.Errorf("smart contract state error: cdn node %v dont exist in the cluser %v", s.ShortState.NodeID, s.ShortState.ClusterID)
	} else {
		err := s.validateFromSC(rawNodesState[s.ShortState.NodeID])
		if err != nil {
			return err
		}
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	// Add new nodes
	for _, n := range rawNodesState {
		if _, ok := s.ClusterMap[n.NodeId]; !ok {
			params, err := bucket.ReadCDNNodeParams(n.Params)
			if err != nil {
				log.WithError(err).Warning("Failed to read node params")
				continue
			}
			s.ClusterMap[n.NodeId] = &GossipState{
				LastState: ShortState{
					NodeID:    n.NodeId,
					ClusterID: s.ShortState.ClusterID,
					Url:       params.Url,
					State:     NA,
					Size:      params.Size,
					Location:  params.Location,
					Updated:   0,
				},
				Signature: StateSignature{
					PublicKey:     []byte(params.PublicKey),
					Scheme:        crypto.Sr25519,
					MultiHashType: 0,
					Timestamp:     0,
				},
				Statistic: Statistic{},
				Checks:    make(map[uint32]Check),
			}
		}
	}
	// Remove not existing Nodes from the list and update params for existing
	for _, n := range s.ClusterMap {
		if _, ok := rawNodesState[n.LastState.NodeID]; ok {
			err := fillGossipStateFromRaw(n, rawNodesState[n.LastState.NodeID])
			if err != nil {
				log.WithError(err).Warning("Failed to update state topologies from smart contract")
				continue
			}
		} else {
			delete(s.ClusterMap, n.LastState.NodeID)
			continue
		}
	}
	return nil
}

func (s *CDNState) updateClusterMapFromGossips() {
	for _, nodeState := range s.ClusterMap {
		// take random chance for check. 1/gossipChance that node will be checked
		r := rand.Intn(s.gossipChance)
		// don't check self
		if r == 0 && nodeState.LastState.NodeID != s.ShortState.NodeID && nodeState.LastState.Url != "" {
			foreignState, err := nodeState.getStateFromURL()
			if err != nil {
				log.WithError(err).Warningf("can't read node %v state", nodeState.LastState.NodeID)
				continue
			}
			s.mergeWithForeignState(foreignState)
		}
	}
}

func (s *CDNState) mergeWithForeignState(foreignState *CDNState) {
	if s.ShortState.NodeID == foreignState.ShortState.NodeID {
		log.Warning("can't merge with self")
		return
	}
	if _, ok := s.ClusterMap[foreignState.ShortState.NodeID]; !ok {
		log.Warningf("can't merge with node %v, not in cluster", foreignState.ShortState.NodeID)
		return
	}
	if foreignState.Signature.MultiHashType != 0 {
		log.Warningf("can't merge with node %v, wrong multyhash type", foreignState.ShortState.NodeID)
		return
	}
	if foreignState.Signature.Timestamp == 0 {
		log.Warningf("can't merge with node %v, no time mark on signature", foreignState.ShortState.NodeID)
		return
	}
	message, _, err := foreignState.ShortState.cidMessage(foreignState.Signature.MultiHashType, foreignState.Signature.Timestamp)
	if err != nil {
		log.WithError(err).Warningf("can't encode node %v state", foreignState.ShortState.NodeID)
		return
	}
	if !foreignState.Signature.verify(message) {
		log.WithError(err).Warningf("can't varify node %v state's signature", foreignState.ShortState.NodeID)
		return
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	// fill simple state in map
	s.ClusterMap[foreignState.ShortState.NodeID].LastState = foreignState.ShortState
	s.ClusterMap[foreignState.ShortState.NodeID].Signature = foreignState.Signature
	s.ClusterMap[foreignState.ShortState.NodeID].Statistic = foreignState.Statistic
	// merge gossips checks if they are fresher
	for _, gossipState := range s.ClusterMap {
		foreignGossipState, ok := foreignState.ClusterMap[gossipState.LastState.NodeID]
		if !ok {
			log.Warningf("can't merge with node %v, not in cluster", gossipState.LastState.NodeID)
			continue
		}
		if foreignGossipState.LastState.NodeID != gossipState.LastState.NodeID {
			log.Warningf("can't merge with node %v, invalid state", foreignGossipState.LastState.NodeID)
			continue
		}
		if !bytes.Equal(foreignGossipState.Signature.PublicKey, gossipState.Signature.PublicKey) {
			log.Warningf("can't merge with node %v, new public key", foreignGossipState.LastState.NodeID)
			continue
		}
		if foreignGossipState.Signature.MultiHashType != 0 {
			log.Warningf("can't merge with node %v, wrong multyhash type", foreignState.ShortState.NodeID)
			continue
		}
		if foreignGossipState.Signature.Timestamp == 0 {
			log.Warningf("can't merge with node %v, no time mark on signature", foreignState.ShortState.NodeID)
			continue
		}

		message, _, err := foreignGossipState.LastState.cidMessage(foreignGossipState.Signature.MultiHashType, foreignGossipState.Signature.Timestamp)
		if err != nil {
			log.WithError(err).Warningf("can't encode node %v state", foreignGossipState.LastState.NodeID)
			continue
		}
		if !foreignGossipState.Signature.verify(message) {
			log.WithError(err).Warningf("can't varify node %v state's signature", foreignGossipState.LastState.NodeID)
			continue
		}

		// take fresher state
		if foreignGossipState.LastState.Updated > gossipState.LastState.Updated {
			gossipState.LastState = foreignGossipState.LastState
			gossipState.Signature = foreignGossipState.Signature
			gossipState.Statistic = foreignGossipState.Statistic
		}
		// merge checks
		for checkerId, fCheck := range foreignGossipState.Checks {
			if _, ok := gossipState.Checks[checkerId]; ok {
				if fCheck.SateCheck.Updated > gossipState.Checks[checkerId].SateCheck.Updated {
					gossipState.Checks[checkerId] = fCheck
				}
			} else {
				gossipState.Checks[checkerId] = fCheck
			}
		}

	}
}

func (ngs GossipState) getStateFromURL() (*CDNState, error) {

	//TODO: Rebuild for use GRPC
	return &CDNState{}, nil
}

func fillGossipStateFromRaw(s *GossipState, r *bucket.CDNNodeStatus) error {
	p, err := bucket.ReadCDNNodeParams(r.Params)
	if err != nil {
		return err
	}
	s.LastState.Location = p.Location
	s.LastState.Url = p.Url
	// Default value
	if p.Size == 0 {
		p.Size = 1
	}
	s.LastState.Size = p.Size
	s.Signature.PublicKey = []byte(p.PublicKey)
	return nil
}

func (s *CDNState) validateFromSC(r *bucket.CDNNodeStatus) error {
	p, err := bucket.ReadCDNNodeParams(r.Params)
	if err != nil {
		return err
	}
	if s.ShortState.Url != p.Url {
		return fmt.Errorf("smart contract state conflict, expected Node URL %v, Configured %v", p.Url, s.ShortState.Url)
	}
	if (s.ShortState.Size != p.Size) && p.Size != 0 {
		return fmt.Errorf("smart contract state conflict, expected Node Size %v, Configured %v", p.Size, s.ShortState.Size)
	}
	b := []byte(p.PublicKey)
	publicKey, err := decodeHex(b)
	if err != nil {
		return fmt.Errorf("can't decode the pyblicKey from smartContract %v not in hex format", p.PublicKey)
	}

	if !bytes.Equal(s.Signature.PublicKey, publicKey) {
		return fmt.Errorf("smart contract state conflict, expected Node Public Key %v, Configured %v", p.PublicKey, s.Signature.PublicKey)
	}
	return nil
}
