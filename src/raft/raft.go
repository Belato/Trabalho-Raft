package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"labrpc"
	"math/rand"
	"sync"
	"time"
)

// import "bytes"
// import "encoding/gob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make().
//
type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

type LogEntry struct {
	Command interface{}
	term    int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2/A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	state RaftState

	// Persistent state on all servers:
	currentTerm int        // latest term server has seen
	votedFor    int        // candidateId that received vote in current term
	logEntries  []LogEntry // each entry contains command for state machine, and term when entry was received by leader

	// Volatile state on all servers:
	commitIndex int // index of highest log entry known to be committed
	lastApplied int // index of highest log entry applied to state machine

	// Volatile state on leaders
	nextIndex  []int // for each server, index of the next log entry to send to that server
	matchIndex []int // for each server, index of highest log entry known to be replicated on server
}

type RaftState int

const (
	Follower RaftState = iota
	Candidate
	Leader
)

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2/A).
	rf.mu.Lock()
	term = rf.currentTerm
	isleader = rf.state == Leader
	rf.mu.Unlock()
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := gob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := gob.NewDecoder(r)
	// d.Decode(&rf.xxx)
	// d.Decode(&rf.yyy)
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2/A, 2B).
	CandidateTerm int
	CandidateId   int
	LastLogIndex  int
	LastLogTerm   int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2/A).
	CurrentTerm int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()

	logLength := len(rf.logEntries)
	lastLogTerm := -1
	lastLogIndex := -1
	if logLength > 0 {
		lastLogIndex = len(rf.logEntries) - 1
		lastLogTerm = rf.logEntries[lastLogIndex].term
	}

	if rf.currentTerm < args.CandidateTerm {
		reply.VoteGranted = false
		rf.state = Follower
	}
	if rf.currentTerm == args.CandidateTerm &&
		(rf.votedFor == -1 || rf.votedFor == args.CandidateId) &&
		(args.LastLogTerm > lastLogTerm || (args.LastLogTerm == lastLogIndex && args.LastLogIndex >= lastLogIndex)) {
		reply.VoteGranted = true
		rf.votedFor = args.CandidateId
	} else {
		reply.VoteGranted = false
	}
	reply.CurrentTerm = rf.currentTerm
	rf.mu.Unlock()

}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int

	PrevLogIndex int
	PrevLogTerm  int

	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool

	ConflictIndex int
	ConflictTerm  int
}

func (rf *Raft) ApprendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	if args.Term > rf.currentTerm {
		rf.state = Follower
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}
	reply.Success = false

	if args.Term == rf.currentTerm {
		if rf.state != Follower {
			rf.state = Follower
			rf.currentTerm = args.Term
			rf.votedFor = -1
		}

		if args.PrevLogIndex == -1 || (args.PrevLogIndex < len(rf.logEntries) && args.PrevLogTerm == rf.logEntries[args.PrevLogIndex].term) {
			reply.Success = true

			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0

			for !((logInsertIndex >= len(rf.logEntries) || newEntriesIndex >= len(args.Entries)) || (rf.logEntries[logInsertIndex].term != args.Entries[newEntriesIndex].term)) {
				logInsertIndex++
				newEntriesIndex++
			}

			if newEntriesIndex < len(args.Entries) {
				rf.logEntries = append(rf.logEntries[:logInsertIndex], args.Entries[newEntriesIndex:]...)
			}

			if args.LeaderCommit > rf.commitIndex {
				if args.LeaderCommit > len(rf.logEntries)-1 {
					rf.commitIndex = len(rf.logEntries) - 1
				} else {
					rf.commitIndex = args.LeaderCommit
				}
			}
		} else {
			if args.PrevLogIndex >= len(rf.logEntries) {
				reply.ConflictIndex = len(rf.logEntries)
				reply.ConflictTerm = -1
			} else {
				reply.ConflictTerm = rf.logEntries[args.PrevLogIndex].term

				index := args.PrevLogIndex

				for index >= 0 && rf.logEntries[index].term == reply.ConflictTerm {
					index--
				}

				reply.ConflictTerm = index + 1
			}
		}

	}

	reply.Term = rf.currentTerm
	rf.mu.Unlock()
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.state = Follower
	rf.commitIndex = 0
	rf.votedFor = -1
	rf.currentTerm = 0
	rf.logEntries = []LogEntry{}
	rf.logEntries = append(rf.logEntries, LogEntry{Command: rf.currentTerm, term: -1})
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	go func() {
		for {
			rf.mu.Lock()

			for rf.commitIndex > rf.lastApplied {
				rf.lastApplied = rf.lastApplied + 1
			}
			rf.mu.Unlock()
			time.Sleep((50 * time.Millisecond))
		}
	}()

	go func() {
		for {
			rf.mu.Lock()
			if rf.state == Follower {
				rf.state = Candidate
			}
			rf.mu.Unlock()

			duration := time.Duration(350 + rand.Intn(100-(-100)+100))
			time.Sleep(duration * time.Millisecond)

			rf.mu.Lock()

			if rf.state == Candidate {
				c := 0
				logLength := len(rf.logEntries)

				lastTerm := 0
				lastIndex := logLength - 1
				requestTerm := rf.currentTerm + 1

				if logLength > 0 {
					lastTerm = rf.logEntries[logLength-1].term
				}

				rvArgs := RequestVoteArgs{requestTerm, rf.me, lastIndex, lastTerm}
				rvReplies := make([]RequestVoteReply, len(rf.peers))

				for index := range rf.peers {
					go func(index int) {
						ok := rf.sendRequestVote(index, &rvArgs, &rvReplies[index])
						rf.mu.Lock()
						if rvReplies[index].CurrentTerm > rf.currentTerm {
							rf.currentTerm = rvReplies[index].CurrentTerm
							rf.state = Follower
						} else if ok && rvArgs.CandidateTerm == rf.currentTerm && rvReplies[index].VoteGranted {
							c++
							if c > len(rf.peers)/2 && rf.state != Leader {
								rf.state = Leader
								rf.currentTerm = requestTerm
								rf.nextIndex = make([]int, len(rf.peers))
								rf.matchIndex = make([]int, len(rf.peers))

								for i := range rf.peers {
									rf.nextIndex[i] = len(rf.logEntries)
								}

							}
						}
						rf.mu.Unlock()
					}(index)
				}
			}

		}

		rf.mu.Unlock()
	}()

	return rf
}
