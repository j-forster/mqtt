
const fs = require('fs');
const EventEmitter = require('events');

///////////////////////////////////////////////////////////////////////////////


var port = 1883;
var host = '127.0.0.1';
const help = `Hi-MQTT Tests and benchmarks
Usage:
 npm run test [-- [-h host] [-p port] [-g grep]]
Args:
 host: mqtt host (default: "127.0.0.1")
 port: mqtt port (default: "1883")
 grep: test specifications (default: "test")
Example:
  Default: (tests only, no benchmarks):
 > npm run test -- -h localhost -p 1883 -g test
  Benchmarks:
 > npm run test -- -g benchmark`

 var config = {};

for(var i=2; i<process.argv.length; i++) {

  if(process.argv[i] == "-help") {
    console.info(help)
    process.exit()
  }

  if(process.argv[i][0] == "-") {
    config[process.argv[i].substr(1)]= process.argv[++i];
    continue
  }

  console.error(`[Error] Unaligned parameter: ${ process.argv[i] }\n${ help }`);
  process.exit(1);
}

var grep = config.g || config.grep || "test";

///////////////////////////////////////////////////////////////////////////////


class Test extends EventEmitter {
  constructor(file) {
    super();

    this.file = file;
  }

  load() {

    console.info(`[Test ] Loading "${ this.file }" ..`);
    require('../test/' + this.file).call(this, config, this);
  }

  exit(code) {

    this.code = code;
    this.emit("complete", code);
  }

  fail() {

    this.exit(1)
  }

  complete() {

    this.exit(0)
  }
}


///////////////////////////////////////////////////////////////////////////////


var tests = fs.readdirSync('test/').filter(file => file.includes(grep));
var completed = 0;

if(tests.length === 0) {

  console.warn(`[Test ] No tests matching: "${ grep }".`);
  process.exit(0)
}

console.warn(`[Test ] Found ${ tests.length } tests.`);


///////////////////////////////////////////////////////////////////////////////



///////////////////////////////////////////////////////////////////////////////

runNextTest()

function runNextTest() {

  if(completed == tests.length) {

    console.log(`[Test ] Completed all ${ completed } tests!`);
    process.exit(0)
  }

  let n = completed;

  if(n != completed)
    return
  completed = n + 1;

  test = new Test(tests[n])
  test.once("complete", (code) => {

    if(code != 0) console.info(`[Test ] "${ test.file }" failed!`);
    else console.info(`[Test ] "${ test.file }" completed!`);

    completed = n + 1;
    runNextTest();
  });
  test.load();
}
