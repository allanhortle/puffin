#!/usr/bin/env node
'use strict';

/**
 * Module dependencies.
 */

var program = require('commander');
var log = require('../lib/logger.js').puffin;
var pkg = require('../package.json');

var install = require('../lib/install.js');


process.title = 'puffin';



program
	.version(pkg.version)
	.option('-p, --peppers', 'Add peppers')
	.option('-P, --pineapple', 'Add pineapple')
	.option('-b, --bbq', 'Add bbq sauce')
	.option('-c, --cheese [type]', 'Add the specified type of cheese [marble]', 'marble');


// $ deploy setup stage
// $ deploy setup
program
  .command('setup [env]')
  .description('run setup commands for all envs')
  .action(function(env){
	env = env || 'all';
	console.log('setup for %s env(s)', env);
  });

program
	.command('install')
	.description('Run brew install commands')
	.action(install);

program
	.command('reload')
	.description('Run brew install commands')
	.action(function(env){

	});

program
	.command('reload')
	.description('Run brew install commands')
	.action(function(env){
		child_process.execFileSync('./bin/brew.sh');
	});



program.parse(process.argv);
