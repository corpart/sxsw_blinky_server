// requires d3!
console.log("loading vtr.sckt!");

var vtr = {};


vtr.sckt = function (wsuri, btnfilename, startvote, endvote) {
  var self = this; // http://stackoverflow.com/questions/20279484/how-to-access-the-correct-this-inside-a-callback

  self.wsuri = wsuri; // websocket uri to connect to

  // set function to push new bubbles
  self.startvote = startvote;
  self.endvote = endvote;

  // open websocket to control server
  self.ws = new WebSocket(self.wsuri);

  self.ws.onopen = function() {
      console.log("connected to " + wsuri);
  }

  self.ws.onclose = function(e) {
      console.log("connection closed (" + e.code + ")");
  }

  // pass messages to handle message method
  self.ws.onmessage = function(e) {
      console.log("message received: " + e.data);
      self.handlemsg(JSON.parse(e.data));
  }

  // load vote station data
  self.stns = {};
  d3.json(btnfilename, function(error, data) {
    data.forEach(function (d) {
      var k = d.source + d.choice;
      self.stns[k] = {
        "id": +d.id,
        "source": d.source,
        "choice": d.choice
      };
    });
  });
}

vtr.sckt.prototype = {

  constructor: vtr.sckt,

  handlemsg: function (msg) {

    // reject messages without a flavor key
    if (! ("flavor" in msg)) {
      console.log("cant handle unexpected message format: " + msg);
      return;
    }

    // only handle start & end touch messages
    if (! (msg.flavor === "start_touch" || msg.flavor === "end_touch")) {
      return;
    }

    // build vote station key & check that key is in station index
    var k = msg.source + msg.choice;
    if (! (k in this.stns)) {
      console.log("cant accept touch from unexpected source: " + msg);
      return;
    }

    // get station id
    var sid = this.stns[k].id;

    // if this is start touch message call start vote
    if (msg.flavor === "start_touch") {
      console.log("starting vote for station: " + sid);
      this.startvote(sid);
      return;
    }

    // otherwise call end vote
    console.log("ending vote for station: " + sid);
    this.endvote(sid);
    return;
  }
}
