--dummy data for postgres

INSERT INTO public.dbprefixsections (
name, abbreviation, "position", hidden, id) VALUES (
'ssss'::text, 'sss'::text, '0'::smallint, false::boolean, 1)
 returning id;

INSERT INTO public.dbprefixboards (
section_id, uri, dir, navbar_position, title, subtitle, description, max_file_size, max_threads, default_style, locked, force_anonymous, autosage_after, no_images_after, max_message_length, min_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog) VALUES (
'1'::bigint, 'drrr'::text, 'dr'::character varying(45), '0'::smallint, 'title'::character varying(45), 'subtitle'::character varying(64), 'descr'::character varying(64), '456456'::integer, '444'::smallint, 'idk'::character varying(45), false::boolean, true::boolean, '33'::smallint, '44'::smallint, '4435'::smallint, '23'::smallint, true::boolean, true::boolean, true::boolean, true::boolean)
 returning id;

INSERT INTO public.dbprefixthreads (
board_id) VALUES (
'1'::bigint)

 returning id;INSERT INTO public.dbprefixthreads (
board_id) VALUES (
'1'::bigint)
 returning id;

 INSERT INTO public.dbprefixposts (
thread_id, is_top_post, ip, message, message_raw, password, created_on) VALUES (
'1'::bigint, true::boolean, '1'::integer, 'ffff'::text, 'ddddd'::text, 'ffff'::text, '2020-04-12 20:51:25.438903')
 returning id;

 INSERT INTO public.dbprefixposts (
thread_id, ip, message, message_raw, password, created_on) VALUES (
'1'::bigint, '1'::integer, 'sss'::text, 'ssss'::text, 'ssss'::text, '2020-04-12 20:51:52.465178')
 returning id;

  INSERT INTO public.dbprefixposts (
thread_id, is_top_post, ip, message, message_raw, password, created_on) VALUES (
'2'::bigint, true::boolean, '1'::integer, 'ffff'::text, 'ddddd'::text, 'ffff'::text, '2020-03-12 20:51:25.438903')
 returning id;

 INSERT INTO public.dbprefixposts (
thread_id, ip, message, message_raw, password, created_on) VALUES (
'2'::bigint, '1'::integer, 'sss'::text, 'ssss'::text, 'ssss'::text, '2020-03-12 20:51:52.465178')
 returning id;

 INSERT INTO public.dbprefixfiles (
post_id, file_order, original_filename, filename, checksum, file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height) VALUES (
'2'::bigint, '0'::integer, '2a'::character varying(255), '2a'::character varying(45), '1'::integer, '1'::integer, false::boolean, '5'::integer, '5'::integer, '10'::integer, '10'::integer)
 returning id;

 INSERT INTO public.dbprefixfiles (
post_id, file_order, original_filename, filename, checksum, file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height) VALUES (
'1'::bigint, '0'::integer, '1a'::character varying(255), '1a'::character varying(45), '2'::integer, '2'::integer, false::boolean, '5'::integer, '5'::integer, '10'::integer, '10'::integer)
 returning id;

 INSERT INTO public.dbprefixfiles (
post_id, file_order, original_filename, filename, checksum, file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height) VALUES (
'3'::bigint, '0'::integer, '3a'::character varying(255), '3a'::character varying(45), '3'::integer, '3'::integer, false::boolean, '5'::integer, '5'::integer, '10'::integer, '10'::integer)
 returning id;

 INSERT INTO public.dbprefixfiles (
post_id, file_order, original_filename, filename, checksum, file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height) VALUES (
'2'::bigint, '1'::integer, '2b'::character varying(255), '2b'::character varying(45), '4'::integer, '4'::integer, false::boolean, '5'::integer, '5'::integer, '10'::integer, '10'::integer)
 returning id;

 INSERT INTO public.dbprefixfiles (
post_id, file_order, original_filename, filename, checksum, file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height) VALUES (
'3'::bigint, '1'::integer, '3b'::character varying(255), '3b'::character varying(45), '5'::integer, '5'::integer, false::boolean, '55'::integer, '5'::integer, '10'::integer, '10'::integer)
 returning id;