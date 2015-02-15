var winston = require('winston');



var logger = new (winston.Logger)({
	transports: [
		new (winston.transports.Console)({
			colorize: true,
		    prettyPrint: true
		})
	],
	levels: {
    	puffin: 1,
    	info: 1,
    	baz: 2,
    	foobar: 3
    },
    colors: {
    	puffin: 'magenta',
    	info: 'green',
    	baz: 'yellow',
    	foobar: 'red'
    }
});

module.exports = logger;
