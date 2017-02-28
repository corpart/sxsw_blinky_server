// requires d3!
console.log("loading Vtr!");

var Vtr = function (filename, bubblefunc) {
  this.msmx = 5000; // max touch duration in milliseconds
  this.rmn = 5; // min radius in pixels
  this.rmx = 15; // max radius in pixels

  // set function to push new bubbles
  this.pushbubble = bubblefunc;

  // load vote station data
  this.stns = {};
  this.tchs = {}  // data structure to track open touches
  d3.json(filename, function(error, data) {

  data.forEach(function(d) {
    var k = d.source + d.choice;
    this.stns[k] = {
      "id": +d.id,
      "source": d.source,
      "choice": d.choice,
      "x3": +d.x3,
      "y3": +d.y3
    };
    this.tchs[k] = -1;
  });

  // open websocket to control server
  this.sckt = new WebSocket("ws://127.0.0.1:8888");
  this.sckt.onopen = function() {
      console.log("connected to " + wsuri);
  }

  this.sckt.onclose = function(e) {
      console.log("connection closed (" + e.code + ")");
  }

  this.sckt.onmessage = function(e) {
      console.log("message received: " + e.data);
      this.handlemsg(e.data);
  }
}

Vtr.prototype = {

  constructor: Vtr,

  handlemsg: function (msg) {

    if (msg.flavor === "start_touch") {
      var k = msg.source + msg.choice;
      if (k in this.tchs) {
        this.tchs[k] = Date.now(); // set timestamp for touch start
      } else {
        console.log("cant start touch from unexpected source: " + msg);
      }

    } else if (msg.flavor === "end_touch") {
      var k = msg.source + msg.choice;
      if (k in this.tchs) {
        if (this.tchs[k] > 0) {
          var ms = Date.now() - this.tchs[k]; // get millis since start touch
          r = this.ms2r(ms);
          stn = this.stns[k]

          // call external pushbubble func
          this.pushbubble(stn.x3, stn.y3, r, stn.id);
          
        } else {
          console.log("cant end unstarted touch: " + msg);
        }
      } else {
        console.log("cant end touch from unexpected source: " + msg);
      }
    }
  },

  ms2r: function(ms) {
    ms = Math.min(ms, this.msmx);
    return this.rmn + (this.rmx - this.rmn) * ms / this.msmx;
  }

}
