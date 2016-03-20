# SignalBouncer

WebRTC signaling server.

### Protocols

- XHR + SSE

### Todos

- Support for XHR Long-Polling
- Support for Websocket
- Move room to external for service immutability

### Endpoints

Subscribe to signal stream, once subscribed it'll send an initial event with `peerId` as the data. After that, future events are all signals.

```
GET /stream/:protocol/:roomName
```

Emit signal:

```
POST /signal/:roomName/:peerId
```

### Example

```javascript
var source = new EventSource('http://127.0.0.1:8080/stream/sse/awesomeroom')
source.onopen = function() {
  console.log('sse connected')
}
source.onerror = function(err) {
  console.log('sse error:', err)
}
source.addEventListener('peerId', function(event) {
  console.log('peerId:', event.data)
})
source.addEventListener('signal', function(event) {
  console.log('signal:', event.data)
})
```
