#!/usr/bin/env node
'use strict';

var program = require('commander');
var util = require('util');
var prompt = require('prompt');

var log = require('../lib/logger.js');
var pkg = require('../package.json');
var install = require('../lib/install.js');
var installApps = require('../lib/installApps.js');
var reload = require('../lib/reload.js');

prompt.message = '';
prompt.delimiter = "";
// prompt.start();

process.title = 'puffin';
program
	.version(pkg.version)
	.option('-f, --force', 'Force');

program
	.command('install')
	.description('Run brew install commands')
	.action(install);

program
	.command('install-apps')
	.description('Run brew install apps commands')
    .action(installApps);

program
	.command('reload')
	.description('rsync the dotfiles')
	.option('-f, --force', 'Force')
	.action(function (cmd, options) {
		if (program.force) {
			reload();
		} else {
			prompt.get([{
				name: 'sure',
			    description: 'Are you sure you want to reload and overwite your dotfiles? (y/N)'
			}], function (err, result) {
				if (err) {
					return console.log('');
				}
				if (result.sure === 'y') {
					reload();
				}
			});
		}


	})
;


program.parse(process.argv);
