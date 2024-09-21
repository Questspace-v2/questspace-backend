ALTER TABLE questspace.task ADD COLUMN media_urls varchar[] CHECK ( cardinality(media_urls) < 6 );

UPDATE questspace.task SET media_urls = ARRAY[media_url]
    WHERE length(media_url) > 0 AND cardinality(media_urls) = 0;
