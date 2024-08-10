-- This should be run only after gochan has been started for the first time 
-- Also this should only be used in a development environment
-- mysql -u gochan -D gochan -pgochan < /vagrant/devtools/mysql_dummydata.sql

INSERT INTO gc_threads (board_id) values(1);
INSERT INTO `gc_posts` (
	`thread_id`,`ip`,`name`,`tripcode`,`email`,`subject`,`message`,`message_raw`
) VALUES (
	1,'192.168.56.1','Name','Tripcode','email@email.com','Subject','Message<br /><b>bold text</b>','Message
[b]bold text[/b]'
);