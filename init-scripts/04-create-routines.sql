create function public.register_user(p_username character varying, p_password_hashed character varying, p_email character varying, p_full_name character varying, p_cif character varying) returns character varying
    language plpgsql
as
$$
declare
    l_user_id users.id%type;
    l_response varchar;
    l_countor integer;
begin
    l_response := validate_cnp(p_cif);
    if l_response <> 'OK' then
        return 'CIF VALIDATION: '||l_response;
    end if;
    if length(p_full_name) =0 or p_full_name is null then
        return 'ERROR - Invalid length of name';
    end if;
    if length(p_username) =0 or p_username is null then
        return 'ERROR - Invalid length of username';
    end if;

    select count(*) into l_countor
    from users where username = upper(p_username);

    if l_countor >0 then
        return 'ERROR - Username already exists!';
    end if;


    insert into users(full_name, username, password_hashed, cif, email)
    values(p_full_name,p_username,p_password_hashed,p_cif,p_email)
    returning id into l_user_id;
--     commit;

    return 'OK';

end;
$$;

alter function public.register_user(varchar, varchar, varchar, varchar, varchar) owner to gogymrest;

create function public.validate_cnp(p_cnp character varying) returns character varying
    language plpgsql
as
$$
declare
    v_weights int[] := array[2,7,9,1,4,6,3,5,8,2,7,9];
    v_sum int := 0;
    v_check_digit int;
    v_calculated_digit int;
    i int;
begin
    -- Check if CNP is null or empty
    if p_cnp is null or trim(p_cnp) = '' then
        return 'ERROR - CNP cannot be null or empty!';
    end if;

    -- Remove any spaces and convert to uppercase
    p_cnp := trim(upper(p_cnp));

    -- Check length (CNP must be exactly 13 digits)
    if length(p_cnp) != 13 then
        return 'ERROR - CNP must be exactly 13 digits!';
    end if;

    -- Check if all characters are digits
    if p_cnp !~ '^[0-9]+$' then
        return 'ERROR - CNP must contain only digits!';
    end if;

    -- Validate first digit (sex and century)
    if substring(p_cnp, 1, 1) not in ('1', '2', '3', '4', '5', '6', '7', '8', '9') then
        return 'ERROR - Invalid sex/century digit!';
    end if;

    -- Validate year (digits 2-3)
    -- Additional validation could be added here for realistic year ranges

    -- Validate month (digits 4-5)
    if substring(p_cnp, 4, 2)::int not between 1 and 12 then
        return 'ERROR - Invalid month!';
    end if;

    -- Validate day (digits 6-7)
    if substring(p_cnp, 6, 2)::int not between 1 and 31 then
        return 'ERROR - Invalid day!';
    end if;

    -- Validate county code (digits 8-9)
    if substring(p_cnp, 8, 2)::int not between 1 and 52 then
        return 'ERROR - Invalid county code!';
    end if;

    -- Calculate check digit using the CNP algorithm
    for i in 1..12 loop
            v_sum := v_sum + (substring(p_cnp, i, 1)::int * v_weights[i]);
        end loop;

    v_calculated_digit := v_sum % 11;

    -- If remainder is 10, check digit should be 1
    if v_calculated_digit = 10 then
        v_calculated_digit := 1;
    end if;

    -- Get the actual check digit (13th digit)
    v_check_digit := substring(p_cnp, 13, 1)::int;

    -- Validate check digit
    if v_check_digit != v_calculated_digit then
        return 'ERROR - Invalid check digit!';
    end if;

    -- If all validations pass
    return 'OK';

exception
    when others then
        return 'ERROR - Invalid CNP format: ' || SQLERRM;
end;
$$;

alter function public.validate_cnp(varchar) owner to gogymrest;

create function public.create_gym(p_name character varying, p_max_people integer, p_max_resevarions integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
declare
    l_gym_id gyms.id%type;
begin
    if length(p_name) = 0 or p_name is null then
        return 'ERROR - Name is invalid!';
    end if;

    insert into gyms(name, members)
    values(p_name,0)
    returning id into l_gym_id;

    insert into gym_stats(gym_id, max_people, max_resevations, current_people,
                          current_reservations, current_combined)
    values(l_gym_id,p_max_people,p_max_resevarions,0,0,0);

    insert into user_gyms(user_id, gym_id, crated_on)
    values(p_user_id,l_gym_id,now());


    return 'OK';
end;
$$;

alter function public.create_gym(varchar, integer, integer, integer) owner to gogymrest;

create function public.add_user_to_gym(p_user_id integer, p_gym_id integer) returns character varying
    language plpgsql
as
$$
declare
    l_countor integer;
begin
    if p_user_id is null then
        return 'ERROR - User is required!';
    end if;

    if p_gym_id is null then
        return 'ERROR - Gym is required!';
    end if;

    select count(*)  into l_countor from user_gyms
    where user_id = p_user_id and gym_id= p_gym_id;

    if l_countor>0 then
        return 'ERROR - User already has access to manage this GYM!';
    end if;


    insert into user_gyms(user_id, gym_id, crated_on)
    values(p_user_id,p_gym_id,now());
    return 'OK';
end;
$$;

alter function public.add_user_to_gym(integer, integer) owner to gogymrest;

create function public.add_user_to_client(p_client_id integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
declare
    l_countor integer;
begin
    if p_user_id is null then
        return 'ERROR - User is required!';
    end if;

    if p_client_id is null then
        return 'ERROR - Gym is required!';
    end if;

    select count(*)  into l_countor from user_clients
    where user_id = p_user_id and client_id= p_client_id;

    if l_countor>0 then
        return 'ERROR - User already has access to manage this Client!';
    end if;


    insert into user_clients(user_id, client_id, created_on)
    values(p_user_id,p_client_id,now());
    return 'OK';
end;
$$;

alter function public.add_user_to_client(integer, integer) owner to gogymrest;

create function public.create_client(p_user_id integer, p_name character varying, p_cif character varying, p_dob date, p_trade_register_no character varying, p_country_id integer, p_state_id integer, p_city character varying, p_street_name character varying, p_street_no character varying, p_building character varying DEFAULT NULL::character varying, p_floor character varying DEFAULT NULL::character varying, p_apartment character varying DEFAULT NULL::character varying) returns character varying
    language plpgsql
as
$$
declare
    l_countor integer;
    L_id_client integer;
begin
    -- Validate user_id
    if p_user_id is null or p_user_id <= 0 then
        return 'ERROR - Valid user ID is required';
    end if;

    -- Check if user exists
    select count(*) into l_countor from users where id = p_user_id;
    if l_countor = 0 then
        return 'ERROR - User does not exist';
    end if;

    -- Validate name
    if p_name is null or length(trim(p_name)) = 0 then
        return 'ERROR - Client name is required';
    end if;

    if length(p_name) > 128 then
        return 'ERROR - Client name cannot exceed 128 characters';
    end if;

    -- Validate CIF (Romanian fiscal code)
    if p_cif is null or length(trim(p_cif)) = 0 then
        return 'ERROR - CIF is required';
    end if;

    if length(p_cif) > 13 then
        return 'ERROR - CIF cannot exceed 13 characters';
    end if;

    -- Check if CIF already exists (unique constraint)
    select count(*) into l_countor from clients where upper(cif) = upper(p_cif);
    if l_countor > 0 then
        return 'ERROR - CIF already exists';
    end if;

    -- Validate date of birth
    if p_dob is null then
        return 'ERROR - Date of birth is required';
    end if;

    -- Check if DOB is not in the future
    if p_dob > current_date then
        return 'ERROR - Date of birth cannot be in the future';
    end if;

    -- Check if DOB is reasonable (not too old, e.g., before 1800)
    if p_dob < date '1800-01-01' then
        return 'ERROR - Date of birth is not valid';
    end if;

    -- Validate trade register number
    if p_trade_register_no is null or length(trim(p_trade_register_no)) = 0 then
        return 'ERROR - Trade register number is required';
    end if;

    if length(p_trade_register_no) > 16 then
        return 'ERROR - Trade register number cannot exceed 16 characters';
    end if;

    -- Validate country_id
    if p_country_id is null or p_country_id <= 0 then
        return 'ERROR - Valid country ID is required';
    end if;

    -- Optional: Check if country exists (uncomment if you have a countries table)
    select count(*) into l_countor from countries where id = p_country_id;
    if l_countor = 0 then
        return 'ERROR - Country does not exist';
    end if;

    -- Validate state_id
    if p_state_id is null or p_state_id <= 0 then
        return 'ERROR - Valid state ID is required';
    end if;

    -- Optional: Check if state exists (uncomment if you have a states table)
    select count(*) into l_countor from states where id = p_state_id and country_id = p_country_id;
    if l_countor = 0 then
        return 'ERROR - State does not exist for the specified country';
    end if;

    -- Validate city
    if p_city is null or length(trim(p_city)) = 0 then
        return 'ERROR - City is required';
    end if;

    if length(p_city) > 64 then
        return 'ERROR - City name cannot exceed 64 characters';
    end if;

    -- Validate street name
    if p_street_name is null or length(trim(p_street_name)) = 0 then
        return 'ERROR - Street name is required';
    end if;

    if length(p_street_name) > 64 then
        return 'ERROR - Street name cannot exceed 64 characters';
    end if;

    -- Validate street number
    if p_street_no is null or length(trim(p_street_no)) = 0 then
        return 'ERROR - Street number is required';
    end if;

    if length(p_street_no) > 16 then
        return 'ERROR - Street number cannot exceed 16 characters';
    end if;

    -- Validate optional fields (building, floor, apartment) - only length checks
    if p_building is not null and length(p_building) > 16 then
        return 'ERROR - Building cannot exceed 16 characters';
    end if;

    if p_floor is not null and length(p_floor) > 8 then
        return 'ERROR - Floor cannot exceed 8 characters';
    end if;

    if p_apartment is not null and length(p_apartment) > 8 then
        return 'ERROR - Apartment cannot exceed 8 characters';
    end if;

    -- Insert the client
    insert into clients(
        name, cif, dob, trade_register_no, country_id, state_id,
        city, street_name, street_no, building, floor, apartment,
        created_on, updated_on, created_by, updated_by
    ) values (
                 trim(p_name), upper(trim(p_cif)), p_dob, trim(p_trade_register_no),
                 p_country_id, p_state_id, trim(p_city), trim(p_street_name),
                 trim(p_street_no), trim(p_building), trim(p_floor), trim(p_apartment),
                 now(), now(), p_user_id, p_user_id
             ) returning id into L_id_client;

    insert into user_clients(user_id, client_id, created_on)
    values(p_user_id,L_id_client,now());
    return 'OK';

exception
    when others then
        return 'ERROR - ' || SQLERRM;
end;
$$;

alter function public.create_client(integer, varchar, varchar, date, varchar, integer, integer, varchar, varchar, varchar, varchar, varchar, varchar) owner to gogymrest;

create function public.add_membership_to_gym(p_membership_id integer, p_gym_id integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
declare
begin
    if p_membership_id is null then
        return 'ERROR - Membership needs to be selected!';
    end if;

    if p_gym_id is null then
        return 'ERROR - GYM needs to be selected!';

    end if;

    insert into membership_gyms(membership_id, gym_id, created_by, updated_by)
    values(p_membership_id,p_gym_id,p_user_id,p_user_id);
    return 'OK';
end;
$$;

alter function public.add_membership_to_gym(integer, integer, integer) owner to gogymrest;

create function public.add_machine_to_gym(p_machine_id integer, p_gym_id integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
declare
begin
    if p_machine_id is null then
        return 'ERROR - Machine needs to be selected!';
    end if;

    if p_gym_id is null then
        return 'ERROR - GYM needs to be selected!';
    end if;

    insert into gym_machines( gym_id, machine_id, created_by, updated_by)
    values(p_gym_id,p_machine_id,p_user_id,p_user_id);

    return 'OK';


end;
$$;

alter function public.add_machine_to_gym(integer, integer, integer) owner to gogymrest;

create function public.add_client_membership(p_client_id integer, p_membership_id integer, p_valid_from date, p_user_id integer) returns character varying
    language plpgsql
as
$$
declare
    l_cursor cursor is select * from memberships where id=p_membership_id;
    l_record memberships%rowtype;

    l_contor integer;
begin
    if p_membership_id is null then
        return 'ERROR - Membership needs to be selected!';
    end if;

    if p_client_id is null then
        return 'ERROR - Client needs to be selected!';
    end if;

    select count(*) into l_contor from client_memberships
    where client_id=p_client_id
      and p_valid_from between starting_from and ending_on
      and status = 'active';

    for l_record in l_cursor loop
            if l_contor > 0 then
                return 'ERROR - Client already has an active membership in this period! ['||p_valid_from ||' - '||p_valid_from+l_record.days_no||']';
            end if;


            insert into client_memberships(client_id, membership_id, starting_from, ending_on,
                                           status, created_by, updated_by)
            values(p_client_id,p_membership_id,p_valid_from,
                   p_valid_from+l_record.days_no,'active',p_user_id,p_user_id);


            return 'OK';
        end loop;

    return 'ERROR - Membership not found!';
end;
$$;

alter function public.add_client_membership(integer, integer, date, integer) owner to gogymrest;

create function public.do_client_pass_in_gym(p_client_id integer, p_gym_id integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
DECLARE
    l_contor INTEGER;
    cu record;
BEGIN
    IF p_client_id IS NULL THEN
        RETURN 'ERROR - Client needs to be selected!';
    END IF;

    IF p_gym_id IS NULL THEN
        RETURN 'ERROR - GYM needs to be selected!';
    END IF;

    SELECT COUNT(*) INTO l_contor
    FROM client_memberships cm
             INNER JOIN membership_gyms mg ON mg.membership_id = cm.membership_id
             INNER JOIN memberships m ON m.id = cm.membership_id
    WHERE cm.client_id = p_client_id
      AND mg.gym_id = p_gym_id
      AND now() BETWEEN cm.starting_from AND cm.ending_on
      AND cm.status = 'active'
      AND m.is_active = true;

    IF l_contor = 0 THEN
        RETURN 'ERROR - Access Denied!';
    END IF;

    FOR cu IN (SELECT * FROM gym_stats WHERE gym_id = p_gym_id)
        LOOP
            IF cu.current_people + 1 > cu.max_people THEN
                RETURN 'ERROR - Currently there isn''t any space available!';
            END IF;
        END LOOP;

    INSERT INTO client_passes (gym_id, client_id, action)
    VALUES (p_gym_id, p_client_id, 'in');

    RETURN 'OK';
END;
$$;

alter function public.do_client_pass_in_gym(integer, integer, integer) owner to gogymrest;

create function public.do_client_check_in_gym(p_client_id integer, p_gym_id integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
DECLARE
    l_contor INTEGER;
    cu record;
BEGIN
    IF p_client_id IS NULL THEN
        RETURN 'ERROR - Client needs to be selected!';
    END IF;

    IF p_gym_id IS NULL THEN
        RETURN 'ERROR - GYM needs to be selected!';
    END IF;

    SELECT COUNT(*) INTO l_contor
    FROM client_memberships cm
             INNER JOIN membership_gyms mg ON mg.membership_id = cm.membership_id
             INNER JOIN memberships m ON m.id = cm.membership_id
    WHERE cm.client_id = p_client_id
      AND mg.gym_id = p_gym_id
      AND now() BETWEEN cm.starting_from AND cm.ending_on
      AND cm.status = 'active'
      AND m.is_active = true;

    IF l_contor = 0 THEN
        RETURN 'ERROR - Access Denied!';
    END IF;

    FOR cu IN (SELECT * FROM gym_stats WHERE gym_id = p_gym_id)
        LOOP
            IF cu.current_combined + 1 > cu.max_people THEN
                RETURN 'ERROR - Currently there isn''t any space available!';
            END IF;
        END LOOP;

    update gym_stats
    set current_people = current_people+1,
        current_combined = current_combined+1
    where gym_id =p_gym_id;

    INSERT INTO client_passes (gym_id, client_id, action, created_by)
    VALUES (p_gym_id, p_client_id, 'in',p_user_id);

    RETURN 'OK';
END;
$$;

alter function public.do_client_check_in_gym(integer, integer, integer) owner to gogymrest;

create function public.do_client_check_out_gym(p_client_id integer, p_gym_id integer, p_user_id integer) returns character varying
    language plpgsql
as
$$
DECLARE
    cu record;
BEGIN
    IF p_client_id IS NULL THEN
        RETURN 'ERROR - Client needs to be selected!';
    END IF;

    IF p_gym_id IS NULL THEN
        RETURN 'ERROR - GYM needs to be selected!';
    END IF;

    FOR cu IN (SELECT * FROM client_passes WHERE gym_id = p_gym_id
                                             and client_id= p_client_id
                                             and action='in'
                                             and trunc(created_on )= trunc(now()))
        LOOP
            update gym_stats
            set current_people = current_people-1,
                current_combined = current_combined-1
            where gym_id =p_gym_id;

            INSERT INTO client_passes (gym_id, client_id, action, created_by)
            VALUES (p_gym_id, p_client_id, 'in',p_user_id);
            RETURN 'OK';

        END LOOP;

    return 'ERROR -  Client never checked in in this gym today!';
END;
$$;

alter function public.do_client_check_out_gym(integer, integer, integer) owner to gogymrest;

