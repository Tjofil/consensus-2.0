// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#pragma once

#include <map>
#include <set>
#include <string>
#include <vector>
#include <cstdint>
#include <cstring>


using namespace std;

/////////////////////////////////////////////////////////////////////////////
// Types
/////////////////////////////////////////////////////////////////////////////

// A processor id is an integer in the range of [0, n-1] where n is the number
// of processors
typedef int t_proc;

// A sequence number is an integer j in the range of [0, m_i-1] identifying the
// j-th event created by the i-th processor where m_i is the number of events of
// processor i.
typedef int t_seq;

// A frame index is the disjoint set number (counting from zero) partitioning
// the local graphs.
typedef int t_frame;

// An event consists of a processor id and a sequence number. Note that sequence
// numbers are not unique. Two different events of two processor may share the
// same sequence number.
typedef pair<t_proc, t_seq> t_event;

// The nil event is an event that represents an invalid event used for
// expressing an undefined state of an event variable.
const t_event nil_event = make_pair(-1, -1);

// Set of events
typedef set<t_event> t_eventset;

// Event vector is a mapping Proc -> Seq defining either a downset
// or upset. Since an event links to the previous event of the same
// processor (if it is not an genesis event), a downset/upset can be
// defined by the highest/lowest sequence number per processor. If
// a pair does not exist for a processor, no events are reachable from/to.
typedef map<t_proc, t_seq> t_eventvector;

/////////////////////////////////////////////////////////////////////////////
// System State
/////////////////////////////////////////////////////////////////////////////

class Lachesis {
private:
  // Number of processors
  int num_processors;

  // Transition step in the simulation
  int step;

  // The parents of an event are represented as a function that maps an event to
  // its parents. parents[e] |-> {e1, ..., ek} where {e1, .., ek} are the
  // parents of the event e.
  map<t_event, t_eventset> parents;

  // The down- and up-sets of events are represented as event vectors, i.e.,
  // downset[e] -> {(p1,s1), (p2,s2), ..., (pk,sk)} defines the frontier of
  // the downset. For all i, events (pi,si') such that si' <= si are reachable
  // in the downset. By definition e=(p,s) is also included in downset[e].
  map<t_event, t_eventvector> downset, upset;

  // Sequence number of the head (most recent) event of a processor in the view
  // of a processor. Function head_seqnum[p1,p2] -> seq defines the most recent
  // event that processor p1 sees that was generated by processor p2.
  vector<vector<t_seq>> head_seqnum;

  // Last decided frame of a processor
  map<t_proc, t_frame> last_decided_frame;

  // Frame index stores the frame index of each event in the local view of a
  // processor.
  vector<map<t_event, t_frame>> frame_idx;

  // Frame roots of a frame indexed by processor and frame number
  vector<vector<t_eventset>> frame_roots;

  // First atropos event for the whole network
  // NB: The first processor finding the first atropos will set the first
  // atropos
  t_event first_atropos = nil_event;

  // Chain of atropos events for the whole network
  map<t_event, t_event> atropos_chain;

  // Most recent atropos event of a processor as a function t_proc -> t_event
  map<t_proc, t_event> head_atropos;

  // Root decision of a processor that is a function t_proc -> t_frame -> t_proc
  // -> bool deciding whether a root node is either an atropos candidate or not.
  // If a root node is not in the map, it is still undecided.
  vector<map<t_frame, map<t_proc, bool>>> root_decision;

  // Votes is a function t_proc -> t_frame -> t_event -> t_proc -> bool that
  // collects votes of a processor for a given frame. A vote is associated with
  // a root node. A vote is a boolean vector assigining each processor either a
  // yes or a no vote.
  vector<map<t_frame, map<t_event, map<t_proc, bool>>>> votes;

  // Stake of a processor
  vector<uint64_t> stake;

  // Total stake of all processors
  uint64_t total_stake;

  // Are we performing a legacy frame calculation
  bool is_legacy_frame_calc;

  // Quorum threshold
  uint64_t quorum;

  // sorted PID according to their stake
  vector<t_proc> sorted_pid;

  /////////////////////////////////////////////////////////////////////////////
  // Dump Facility
  /////////////////////////////////////////////////////////////////////////////

  // convert an event to a string
  string event_to_string(t_event e);

  // dump state
  void dump_state();

  // dump downsets and upsets of events
  void dump_vectors(string filename);

  /////////////////////////////////////////////////////////////////////////////
  // Assertions / Safety Property checks
  /////////////////////////////////////////////////////////////////////////////

  // check whether a processor's id is correct.
  void check_procid(t_proc id);

  // check whether an event is semantically correct
  void check_event(t_event a);

  // check whether the proposed atropos event of processor pid
  // is consistent and update head atropos.
  void check_atropos(t_proc pid, t_event atropos);

  // check whether the proposed root event is consistent among all processors,
  // there must not exist a different event of the same processors in another
  // local view for the same frame.
  void check_frame(t_frame frame, t_event new_event);

  /////////////////////////////////////////////////////////////////////////////
  // Block Generation
  /////////////////////////////////////////////////////////////////////////////

  // Recursive procedure to update the upsets of parents
  // the first update of a processor is the smallest seqnum
  // and hence we can stop the update as soon as we see
  // that an intermediate or immediate predecessors that has
  // already a sequence number for the processor.
  void update_upset(t_event new_event, t_event parent_event);

  // Join two event vectors for downset calculations
  // compute max for overlap otherwise use the single value
  // either in a or in b.
  t_eventvector join_downset(const t_eventvector &a, const t_eventvector &b);

  // Forkless cause predicate with unit stake for processors
  // assuming that there are no forks.
  bool forkless_cause(t_event a, t_event b);

  /////////////////////////////////////////////////////////////////////////////
  // Consensus
  /////////////////////////////////////////////////////////////////////////////

  // Choose an atropos event
  void choose_atropos(t_proc pid);

  // Perform voting
  void perform_voting(t_proc pid, t_event new_root);

  // Perform aggregation
  void perform_aggregation(t_proc pid, t_event new_root);

  // Obtain the maximum frame index of all parents
  t_frame get_max_parent_frame(t_proc pid, t_event new_event);

  // Determine if a quorum of roots in a frame can be forkless cause by new
  // event
  bool forkless_cause_on_quorum(t_proc pid, t_frame frame, t_event new_event);

  // Insert root into the frame_roots data structure and increases size if
  // needed
  void insert_frame_root(t_proc pid, t_frame frame, t_event new_event);

  // Update atropos event if the new root event is an atropos event
  void update_atropos(t_proc pid, t_event new_root);

  // Assign a frame index to a new event. If the new event becomes
  // a new frame root, the function returns true; otherwise false.
  // In case the new event is a frame root, we update the frame-
  // root data-structure.
  bool update_frame(t_proc pid, t_event new_event);

  bool update_frame_legacy(t_proc pid, t_event new_event);

  // Update frames and atropos for a newly created/received event
  void update_frame_atropos(t_proc pid, t_event new_event);

  // Check if a root event is an atropos event
  bool is_atropos(t_proc pid, t_event event);

public:
  /////////////////////////////////////////////////////////////////////////////
  // State transition and access methods of the Lachesis protocol
  /////////////////////////////////////////////////////////////////////////////

  // Initialisation of system state
  Lachesis(int n, vector<uint64_t> s, bool legacy = false);
  virtual ~Lachesis() {}

  // Create a new event in processor "producer"
  void create_event(t_proc producer, const vector<t_proc> &parent_processors);

  // Receive the next event from processor "sender" in processor "receiver"
  void receive_event(t_proc receiver, t_proc sender);

  // Receive the next event from processor "sender" in processor "receiver"
  // until sequence number is reached
  void receive_event(t_proc receiver, t_proc sender, t_seq seqnum);

  // Check if a root event is a frame root in a processor
  bool is_frame_root(t_proc pid, t_event event);

  // Check new atropos event for correctness
  bool check_subsequent_atropos(t_event prev_atropos, t_event current_atropos);

  // Check first atropos event of a processor
  bool check_first_atropos(t_event atropos);

  // Dump DAG of processor
  void dump(t_proc pid, string filename);

  // Get frame index
  inline t_frame get_frame(t_proc pid, t_event event) {
    return frame_idx[pid][event];
  }
};
