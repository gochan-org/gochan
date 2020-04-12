/*Gets post-id with thread top post id pairs*/
select 
	posts.id as selfid,
	topposts.id as toppostid
from
	dbprefixposts as posts
	join dbprefixthreads as threads on threads.id = posts.thread_id
	join dbprefixposts as topposts on threads.id = topposts.thread_id
where 
	topposts.is_top_post = TRUE
	
/*The top level files per post*/
	
SELECT files.post_id, filename
	FROM dbprefixfiles as files
	JOIN 
		(SELECT post_id, min(file_order) as file_order
		FROM dbprefixfiles
		GROUP BY post_id) as topfiles 
		ON files.post_id = topfiles.post_id AND files.file_order = topfiles.file_order