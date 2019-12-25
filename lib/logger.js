const chalk = require('chalk');
const log = (color, name) => (...args) => console.log(chalk[color](`[${name}]`), ...args);

module.exports = {
    puffin: log('magenta', 'puffin'),
    success: log('green', 'success'),
    error: log('red', 'error')
};
