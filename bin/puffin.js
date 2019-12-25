#!/usr/bin/env node
'use strict';

var program = require('commander');

var pkg = require('../package.json');
var install = require('../lib/install.js');
var installApps = require('../lib/installApps.js');
var reload = require('../lib/reload.js');


process.title = 'puffin';
program
	.version(pkg.version)

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
	.action(reload);

program.parse(process.argv);
