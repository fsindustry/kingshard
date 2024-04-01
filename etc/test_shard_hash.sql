# Node 3306
create database if not exists kingshard;
use kingshard;
CREATE TABLE `test_shard_hash_0000`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `test_shard_hash_0001`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `test_shard_hash_0002`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `test_shard_hash_0003`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

# Node 3307
create database if not exists kingshard;
use kingshard;
CREATE TABLE `test_shard_hash_0004`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `test_shard_hash_0005`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `test_shard_hash_0006`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `test_shard_hash_0007`
(
    `id`  bigint(64) unsigned NOT NULL,
    `str` varchar(256) DEFAULT NULL,
    `f` double DEFAULT NULL,
    `e`   enum('test1','test2') DEFAULT NULL,
    `u`   tinyint(3) unsigned DEFAULT NULL,
    `i`   tinyint(4) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;


# insert data from kingshard
insert into test_shard_hash(id,str,f,e,u,i) values(15,"flike",3.14,'test2',2,3);
insert into test_shard_hash(id,str,f,e,u,i) values(7,"chen",2.1,'test1',32,3);
insert into test_shard_hash(id,str,f,e,u,i) values(17,"github",2.5,'test1',32,3);
insert into test_shard_hash(id,str,f,e,u,i) values(18,"kingshard",7.3,'test1',32,3);