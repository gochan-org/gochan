CREATE OR REPLACE FUNCTION ip_cmp(ip1 inet, ip2 inet)
RETURNS int4 AS $$
	BEGIN
		return CASE
			WHEN family(ip1) <> family(ip2) THEN NULL -- comparing an IPv4 to an IPv6 or vice versa
			WHEN ip1 < ip2 THEN -1
			WHEN ip1 > ip2 THEN 1
			ELSE 0
		END;
	END;
$$ LANGUAGE plpgsql IMMUTABLE;