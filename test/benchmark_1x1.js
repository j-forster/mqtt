
const MQTT = require("./MQTT")

///////////////////////////////////////////////////////////////////////////////

var head = Buffer.from([

  // fixed header:
  0x30, // PUBLISH (qos 0)
  0x80 | 0x05, 0x08, // remaining length (1029 bytes)

  // variable header:
  0x00, 0x03, // topic name length
  0x61, 0x2f, 0x62, // topic "a/b"
]);


var payload = Buffer.alloc(1024);
for(var i=0; i<1024; i++)
  payload[i] = Math.random() * 0xff;

var PUBLISH = Buffer.concat([head, payload]);

//

var SUBSCRIBE = Buffer.from([

  // fixed header:
  0x82, // SUBSCRIBE (qos 1)
  0x08, // remaining length (8 bytes)

  // variable header:
  0x12, 0x34, // message id

  0x00, 0x03, // topic name length
  0x61, 0x2f, 0x62, // topic "a/b"
  0x00 // qos 0
]);


var SUBACK = Buffer.from([

  // fixed header:
  0x90, // SUBACK
  0x03, // remaining length (4 bytes)

  // variable header:
  0x12, 0x34, // message id

  // payload:
  0x00 // granted qos (0)
]);


///////////////////////////////////////////////////////////////////////////////


module.exports = (config, test) => {

  var begin;

  //

  var subscriber = new MQTT(config)

  subscriber.on('close', (err) => test.exit(err));

  subscriber.connect(() => {

    subscriber.write(SUBSCRIBE);
    subscriber.once('data', (data) => {

      if(!data.equals(SUBACK)) {

        console.error(`[MQTT ] Error: Unexpected SUBACK message!\nExpected:`, SUBACK, `\nFound:`, data);
        test.fail();
      } else {

        console.log(`[MQTT ] Subscribed. (recieved SUBACK)`);


        var count = config.c || config.count || 100;
        var preload = config.l || config.preload || 10;
        console.log(`[Bench] Network: single publisher -> single subscriber`);
        console.log(`[Bench] Sending ${ count } packages, ${ preload } concurrent ..`);

        var pending = 0;

        var total = count*(8+1024);
        var recieved = 0;
        var n = 0;

        subscriber.on('data', (data) => {

          recieved += data.length;
          total -= data.length;
          while(recieved > (8+1024)) {
            recieved -= 8+1024;
            pending--;
            // process.stdout.write('\b \b');

            if(++n >= count) {

              var end = process.hrtime();
              var delta = (end[0]-begin[0])*1e3+(end[1]-begin[1])/1e6;

              console.log(`[Bench] Completed.`);
              console.log(`[Bench] Time delta: ${ delta } ms`);
              console.log(`[Bench] Msg/Sec: ${ count/delta*1e3 } Msg/s`);

              test.complete();
              return
            }
          }

          if(pending<preload) {
            publisher.write(PUBLISH);
            // process.stdout.write('.');
            pending++;
          }
        });

        //

        var publisher = new MQTT(config)

        publisher.on('close', (err) => test.exit(err));

        publisher.connect(() => {

          console.log(`[Bench] Please wait ..`);

          begin = process.hrtime();

          while(pending<preload) {
            publisher.write(PUBLISH);
            // process.stdout.write('.');
            pending++;
          }
        });




        //

        // test.complete();
      }
    });
  });
}
