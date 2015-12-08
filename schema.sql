--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: pltcl; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS pltcl WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION pltcl; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pltcl IS 'PL/Tcl procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: books; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE books (
    id integer NOT NULL,
    title text,
    author text,
    description text,
    imageurl text,
    isbn_10 character(10),
    isbn_13 character(13)
);


ALTER TABLE public.books OWNER TO appuser;

--
-- Name: books_id_seq; Type: SEQUENCE; Schema: public; Owner: appuser
--

CREATE SEQUENCE books_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.books_id_seq OWNER TO appuser;

--
-- Name: books_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: appuser
--

ALTER SEQUENCE books_id_seq OWNED BY books.id;


--
-- Name: bookstocourse; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE bookstocourse (
    book_id integer,
    course_id integer
);


ALTER TABLE public.bookstocourse OWNER TO appuser;

--
-- Name: posts; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE posts (
    id integer NOT NULL,
    oldversionid integer,
    newversionid integer,
    creatorid integer,
    content text,
    createdate timestamp with time zone,
    enddate timestamp with time zone
);


ALTER TABLE public.posts OWNER TO appuser;

--
-- Name: comments; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE comments (
    parent_listing_id integer
)
INHERITS (posts);


ALTER TABLE public.comments OWNER TO appuser;

--
-- Name: courses; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE courses (
    id integer NOT NULL,
    old_id integer,
    new_id integer,
    creatorid integer,
    title text,
    description text
);


ALTER TABLE public.courses OWNER TO appuser;

--
-- Name: courses_id_seq; Type: SEQUENCE; Schema: public; Owner: appuser
--

CREATE SEQUENCE courses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.courses_id_seq OWNER TO appuser;

--
-- Name: courses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: appuser
--

ALTER SEQUENCE courses_id_seq OWNED BY courses.id;


--
-- Name: directmessages; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE directmessages (
    parent_listing_id integer
)
INHERITS (posts);


ALTER TABLE public.directmessages OWNER TO appuser;

--
-- Name: emailvalidation; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE emailvalidation (
    userid integer,
    validationtoken character varying(100),
    isvalidated boolean
);


ALTER TABLE public.emailvalidation OWNER TO appuser;

--
-- Name: flags; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE flags (
    postid integer,
    reporteduserid integer
)
INHERITS (posts);


ALTER TABLE public.flags OWNER TO appuser;

--
-- Name: listings; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE listings (
    title character varying(255),
    price money,
    condition text
)
INHERITS (posts);


ALTER TABLE public.listings OWNER TO appuser;

--
-- Name: listings_cache; Type: TABLE; Schema: public; Owner: postgres; Tablespace: 
--

CREATE TABLE listings_cache (
    id integer,
    oldversionid integer,
    newversionid integer,
    creatorid integer,
    content text,
    createdate timestamp with time zone,
    enddate timestamp with time zone,
    title character varying(255),
    price money,
    condition text
);


ALTER TABLE public.listings_cache OWNER TO postgres;

--
-- Name: posts_id_seq; Type: SEQUENCE; Schema: public; Owner: appuser
--

CREATE SEQUENCE posts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.posts_id_seq OWNER TO appuser;

--
-- Name: posts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: appuser
--

ALTER SEQUENCE posts_id_seq OWNED BY posts.id;


--
-- Name: professorstocourses; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE professorstocourses (
    prof_id integer,
    course_id integer
);


ALTER TABLE public.professorstocourses OWNER TO appuser;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE sessions (
    userid integer,
    cookieinfo character varying(100),
    valid_til timestamp with time zone
);


ALTER TABLE public.sessions OWNER TO appuser;

--
-- Name: users; Type: TABLE; Schema: public; Owner: appuser; Tablespace: 
--

CREATE TABLE users (
    id integer NOT NULL,
    handle character varying(100),
    email character varying(255),
    passwordbcrypt bytea,
    biography character varying(1000),
    imageurl text,
    ismod boolean,
    isfaculty boolean,
    issuspended boolean,
    isbanned boolean
);


ALTER TABLE public.users OWNER TO appuser;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: appuser
--

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_id_seq OWNER TO appuser;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: appuser
--

ALTER SEQUENCE users_id_seq OWNED BY users.id;


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY books ALTER COLUMN id SET DEFAULT nextval('books_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY comments ALTER COLUMN id SET DEFAULT nextval('posts_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY courses ALTER COLUMN id SET DEFAULT nextval('courses_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY directmessages ALTER COLUMN id SET DEFAULT nextval('posts_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY flags ALTER COLUMN id SET DEFAULT nextval('posts_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY listings ALTER COLUMN id SET DEFAULT nextval('posts_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY posts ALTER COLUMN id SET DEFAULT nextval('posts_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: appuser
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- Name: public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM postgres;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- PostgreSQL database dump complete
--

