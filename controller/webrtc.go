package controller

import (
	"encoding/json"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

// Add to list of tracks and fire renegotation for all PeerConnections
func (room *Room) addTrack(t *webrtc.TrackRemote, clientID int) *webrtc.TrackLocalStaticRTP {
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		room.signalPeerConnections()
	}()

	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	room.trackLocals[t.ID()] = trackLocal
	room.tidTable[clientID] = t.ID()
	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections
func (room *Room) removeTrack(t *webrtc.TrackLocalStaticRTP, clientID int) {
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		room.signalPeerConnections()
	}()

	delete(room.trackLocals, t.ID())
	delete(room.tidTable, clientID)
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks
func (room *Room) signalPeerConnections() {
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		room.dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for _, client := range room.clients {
			if client.pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
				room.unregister <- client
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
			existingSenders := map[string]bool{}

			for _, sender := range client.pc.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := room.trackLocals[sender.Track().ID()]; !ok {
					if err := client.pc.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range client.pc.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range room.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := client.pc.AddTrack(room.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := client.pc.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = client.pc.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}

			client.send <- &Message{
				Action:  OfferAction,
				Content: string(offerString),
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				room.signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
func (room *Room) dispatchKeyFrame() {
	room.listLock.Lock()
	defer room.listLock.Unlock()

	for _, client := range room.clients {
		for _, receiver := range client.pc.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = client.pc.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}
