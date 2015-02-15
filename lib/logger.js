var winston = require('winston');
winston.emitErrs = true;

var logger = new winston.Logger({
    transports: [
        new winston.transports.Console({
            level: 'puffin',
            handleExceptions: true,
            json: false,
            prettyPrint: true,
            colorize: true
        })
    ],
    levels: {
    	puffin: 1,
    	success: 1,
    	error: 5
    },
    colors: {
    	puffin: 'magenta',
    	success: 'green',
    	error: 'red'
    },
    exitOnError: false
});

module.exports = logger;
