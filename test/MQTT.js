const net = require('net');
const EventEmitter = require('events');

var num_client = 1;

const CONNECT = Buffer.from([

  // fixed header:
  0x10, // CONNECT
  0x1d, // remaining length (29 bytes)

  // variable header:
  0x00, 0x06, // protocol name length
  0x4d, 0x51, 0x49, 0x73, 0x64, 0x70, // protocol name "MQIsdp"
  0x03, // mqtt protocol version 3
  0x01, // connect flags (clean session: 1)
  0x00, 0x3c, // keep alive timer: 60s

  // payload
  0x00, 0x0f, // client id length
  0x48, 0x49, 0x4d, 0x51, 0x54, 0x54, 0x2d, 0x54, 0x65, 0x73, 0x74, // client id "HIMQTT-Test"
  0x00, 0x00, 0x00, 0x00 // id appendix (num_client as hex)
]);

const CONNACK = Buffer.from([

  // fixed header:
  0x20, // CONNACK
  0x02, // remaining length (2 bytes)

  // variable header:
  0x00, 0x00, // return code 0 (Connection Accepted)
]);

module.exports = class MQTT extends EventEmitter {

  constructor(config) {
    super();
    this.port = config.p || config.port || '1883';
    this.host = config.h ||  config.host || '127.0.0.1';
    this.conn = new net.Socket();
  }

  write(data) {

    this.conn.write(data);
  }

  connect(cb) {

    this.on('connect', cb);
    console.log(`[TCP  ] Connecting ${ this.host }:${ this.port } ..`);

    var address = "";

    this.conn.connect(this.port, this.host, () => {

      address = this.conn.address().address+':'+this.conn.address().port;

      console.log(`[TCP  ] Connected. ${  address }`);
      console.log(`[MQTT ] Connecting (sending CONNECT msg) ..`);

      if(num_client == 0xffffffff)
        num_client = 1;
      else
        ++num_client;

      CONNECT.write('0000', 27);
      CONNECT.write((num_client).toString(16), 27)

      this.conn.write(CONNECT);

      this.conn.once('data', (data) => {

        if(!data.equals(CONNACK)) {
          console.error(`[MQTT ] Error: Unexpected CONNACK message!\nExpected:`, CONNACK, `\nFound:`, data);
          this.emit('error', 'Unexpected CONNACK message!')
        } else {
          console.log(`[MQTT ] Connected. (recieved CONNACK)`);

          this.conn.on('data', this.emit.bind(this, 'data'));
          this.emit('connect', this.conn);
        }
      });
    });

    this.conn.on('error', (err) => {

      console.error(`[TCP  ] Connection ${ address } error:`, err);
      this.emit('error', err);
    });

    this.conn.on('close', (err) => {

      console.error(`[TCP  ] Connection ${ address } closed.`, err?'(with error)':'');
      this.emit('close', err);
    });
  }

  end() {
    this.conn.end();
  }
}
