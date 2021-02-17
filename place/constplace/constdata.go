//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package constplace stores zettel inside the executable.
package constplace

import (
	"fmt"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

const (
	syntaxTemplate = "mustache"
)

var constZettelMap = map[id.Zid]constZettel{
	id.ConfigurationZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Runtime Configuration",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityOwner,
			meta.KeySyntax:     meta.ValueSyntaxNone,
		},
		"",
	},

	id.BaseTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Base HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(
			`<!DOCTYPE html>
<html{{#Lang}} lang="{{Lang}}"{{/Lang}}>
<head>
<meta charset="utf-8">
<meta name="referrer" content="same-origin">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="generator" content="Zettelstore">
{{{MetaHeader}}}
<link rel="stylesheet" href="{{{StylesheetURL}}}">
<title>{{Title}}</title>
</head>
<body>
<nav class="zs-menu">
<a href="{{{HomeURL}}}">Home</a>
{{#WithUser}}
<div class="zs-dropdown">
<button>User</button>
<nav class="zs-dropdown-content">
{{#WithAuth}}
{{#UserIsValid}}
<a href="{{{UserZettelURL}}}">{{UserIdent}}</a>
{{/UserIsValid}}
{{^UserIsValid}}
<a href="{{{LoginURL}}}">Login</a>
{{/UserIsValid}}
{{/WithAuth}}
{{#CanReload}}
<a href="{{{ReloadURL}}}">Reload</a>
{{/CanReload}}
{{#WithAuth}}
{{#UserIsValid}}
<a href="{{{UserLogoutURL}}}">Logout</a>
{{/UserIsValid}}
{{/WithAuth}}
</nav>
</div>
{{/WithUser}}
<div class="zs-dropdown">
<button>Lists</button>
<nav class="zs-dropdown-content">
<a href="{{{ListZettelURL}}}">List Zettel</a>
<a href="{{{ListRolesURL}}}">List Roles</a>
<a href="{{{ListTagsURL}}}">List Tags</a>
</nav>
</div>
{{#CanCreate}}
<div class="zs-dropdown">
<button>New</button>
<nav class="zs-dropdown-content">
{{#NewZettelLinks}}
<a href="{{{URL}}}">{{Text}}</a>
{{/NewZettelLinks}}
</nav>
</div>
{{/CanCreate}}
<form action="{{{SearchURL}}}">
<input type="text" placeholder="Search.." name="s">
</form>
</nav>
<main class="content">
{{{Content}}}
</main>
{{#FooterHTML}}
<footer>
{{{FooterHTML}}}
</footer>
{{/FooterHTML}}
</body>
</html>`,
		),
	},

	id.LoginTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Login Form HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(
			`<article>
<header>
<h1>{{Title}}</h1>
</header>
{{#Retry}}
<div class="zs-indication zs-error">Wrong user name / password. Try again.</div>
{{/Retry}}
<form method="POST" action="?_format=html">
<div>
<label for="username">User name</label>
<input class="zs-input" type="text" id="username" name="username" placeholder="Your user name.." autofocus>
</div>
<div>
<label for="password">Password</label>
<input class="zs-input" type="password" id="password" name="password" placeholder="Your password..">
</div>
<input class="zs-button" type="submit" value="Login">
</form>
</article>`,
		)},

	id.ListTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore List Meta HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(
			`<h1>{{Title}}</h1>
<ul>
{{#Metas}}<li><a href="{{{URL}}}">{{{Title}}}</a></li>
{{/Metas}}</ul>
{{#HasPrevNext}}
<p>
{{#HasPrev}}
<a href="{{{PrevURL}}}" rel="prev">Prev</a>
{{#HasNext}},{{/HasNext}}
{{/HasPrev}}
{{#HasNext}}
<a href="{{{NextURL}}}" rel="next">Next</a>
{{/HasNext}}
</p>
{{/HasPrevNext}}`)},

	id.DetailTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Detail HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(
			`<article>
<header>
<h1>{{{HTMLTitle}}}</h1>
<div class="zs-meta">
{{#CanWrite}}<a href="{{{EditURL}}}">Edit</a> &#183;{{/CanWrite}}
{{Zid}} &#183;
<a href="{{{InfoURL}}}">Info</a> &#183;
(<a href="{{{RoleURL}}}">{{RoleText}}</a>)
{{#HasTags}}&#183; {{#Tags}} <a href="{{{URL}}}">{{Text}}</a>{{/Tags}}{{/HasTags}}
{{#CanCopy}}&#183; <a href="{{{CopyURL}}}">Copy</a>{{/CanCopy}}
{{#CanFolge}}&#183; <a href="{{{FolgeURL}}}">Folge</a>{{/CanFolge}}
{{#FolgeRefs}}<br>Folge: {{{FolgeRefs}}}{{/FolgeRefs}}
{{#PrecursorRefs}}<br>Precursor: {{{PrecursorRefs}}}{{/PrecursorRefs}}
{{#HasExtURL}}<br>URL: <a href="{{{ExtURL}}}"{{{ExtNewWindow}}}>{{ExtURL}}</a>{{/HasExtURL}}
</div>
</header>
{{{Content}}}
{{#HasBackLinks}}
<details>
<summary>Links to this zettel</summary>
<ul>
{{#BackLinks}}
<li><a href="{{{URL}}}">{{Text}}</a></li>
{{/BackLinks}}
</ul>
</details>
{{/HasBackLinks}}
</article>`)},

	id.InfoTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Info HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(
			`<article>
<header>
<h1>Information for Zettel {{Zid}}</h1>
<a href="{{{WebURL}}}">Web</a>
{{#CanWrite}} &#183; <a href="{{{EditURL}}}">Edit</a>{{/CanWrite}}
{{#CanFolge}} &#183; <a href="{{{FolgeURL}}}">Folge</a>{{/CanFolge}}
{{#CanCopy}} &#183; <a href="{{{CopyURL}}}">Copy</a>{{/CanCopy}}
{{#CanRename}}&#183; <a href="{{{RenameURL}}}">Rename</a>{{/CanRename}}
{{#CanDelete}}&#183; <a href="{{{DeleteURL}}}">Delete</a>{{/CanDelete}}
</header>
<h2>Interpreted Meta Data</h2>
<table>{{#MetaData}}<tr><td>{{Key}}</td><td>{{{Value}}}</td></tr>{{/MetaData}}</table>
{{#HasLinks}}
<h2>References</h2>
{{#HasLocLinks}}
<h3>Local</h3>
<ul>
{{#LocLinks}}
<li><a href="{{{.}}}">{{.}}</a></li>
{{/LocLinks}}
</ul>
{{/HasLocLinks}}
{{#HasExtLinks}}
<h3>External</h3>
<ul>
{{#ExtLinks}}
<li><a href="{{{.}}}"{{{ExtNewWindow}}}>{{.}}</a></li>
{{/ExtLinks}}
</ul>
{{/HasExtLinks}}
{{/HasLinks}}
<h2>Parts and format</h3>
<table>
{{#Matrix}}
<tr>
{{#Elements}}{{#HasURL}}<td><a href="{{{URL}}}">{{Text}}</td>{{/HasURL}}{{^HasURL}}<th>{{Text}}</th>{{/HasURL}}
{{/Elements}}
</tr>
{{/Matrix}}
</table>
</article>`),
	},

	id.FormTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Form HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		`<article>
<header>
<h1>{{Heading}}</h1>
</header>
<form method="POST">
<div>
<label for="title">Title</label>
<input class="zs-input" type="text" id="title" name="title" placeholder="Title.." value="{{MetaTitle}}" autofocus>
</div>
<div>
<div>
<label for="role">Role</label>
<input class="zs-input" type="text" id="role" name="role" placeholder="role.." value="{{MetaRole}}">
</div>
<label for="tags">Tags</label>
<input class="zs-input" type="text" id="tags" name="tags" placeholder="#tag" value="{{MetaTags}}">
</div>
<div>
<label for="meta">Metadata</label>
<textarea class="zs-input" id="meta" name="meta" rows="4" placeholder="metakey: metavalue">
{{#MetaPairsRest}}
{{Key}}: {{Value}}
{{/MetaPairsRest}}
</textarea>
</div>
<div>
<label for="syntax">Syntax</label>
<input class="zs-input" type="text" id="syntax" name="syntax" placeholder="syntax.." value="{{MetaSyntax}}">
</div>
<div>
{{#IsTextContent}}
<label for="content">Content</label>
<textarea class="zs-input zs-content" id="meta" name="content" rows="20" placeholder="Your content..">{{Content}}</textarea>
{{/IsTextContent}}
</div>
<input class="zs-button" type="submit" value="Submit">
</form>
</article>`,
	},

	id.RenameTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Rename Form HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		`<article>
<header>
<h1>Rename Zettel {{.Zid}}</h1>
</header>
<p>Do you really want to rename this zettel?</p>
<form method="POST">
<div>
<label for="newid">New zettel id</label>
<input class="zs-input" type="text" id="newzid" name="newzid" placeholder="ZID.." value="{{Zid}}" autofocus>
</div>
<input type="hidden" id="curzid" name="curzid" value="{{Zid}}">
<input class="zs-button" type="submit" value="Rename">
</form>
<dl>
{{#MetaPairs}}
<dt>{{Key}}:</dt><dd>{{Value}}</dd>
{{/MetaPairs}}
</dl>
</article>`,
	},

	id.DeleteTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Delete HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		`<article>
<header>
<h1>Delete Zettel {{Zid}}</h1>
</header>
<p>Do you really want to delete this zettel?</p>
<dl>
{{#MetaPairs}}
<dt>{{Key}}:</dt><dd>{{Value}}</dd>
{{/MetaPairs}}
</dl>
<form method="POST">
<input class="zs-button" type="submit" value="Delete">
</form>
</article>
{{end}}`,
	},

	id.RolesTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore List Roles HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		`<h1>Currently used roles</h1>
<ul>
{{#Roles}}<li><a href="{{{URL}}}">{{Text}}</a></li>
{{/Roles}}</ul>`,
	},

	id.TagsTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore List Tags HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		`<h1>Currently used tags</h1>
<div class="zs-meta">
<a href="{{{#ListTagsURL}}}">All</a>{{#MinCounts}}, <a href="{{{URL}}}">{{Count}}</a>{{/MinCounts}}
</div>
{{#Tags}} <a href="{{{URL}}}" style="font-size:{{Size}}%">{{Name}}</a><sup>{{Count}}</sup>
{{/Tags}}`,
	},

	id.BaseCSSZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Base CSS",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityPublic,
			meta.KeySyntax:     "css",
		},
		`/* Default CSS */
*,*::before,*::after {
  box-sizing: border-box;
}
html {
  font-size: 1rem;
  font-family: serif;
  scroll-behavior: smooth;
  height: 100%;
}
body {
  margin: 0;
  min-height: 100vh;
  text-rendering: optimizeSpeed;
  line-height: 1.4;
  overflow-x: hidden;
  background-color: #f8f8f8 ;
  height: 100%;
}
nav.zs-menu {
  background-color: hsl(210, 28%, 90%);
  overflow: auto;
  white-space: nowrap;
  font-family: sans-serif;
  padding-left: .5rem;
}
nav.zs-menu > a {
  float:left;
  display: inline-block;
  text-align: center;
  padding:.41rem .5rem;
  text-decoration: none;
  color:black;
}
nav.zs-menu > a:hover, .zs-dropdown:hover button {
  background-color: hsl(210, 28%, 80%);
}
nav.zs-menu form {
  float: right;
}
nav.zs-menu form input[type=text] {
  padding: .12rem;
  border: none;
  margin-top: .25rem;
  margin-right: .5rem;
}
.zs-dropdown {
  float: left;
  overflow: hidden;
}
.zs-dropdown > button {
  font-size: 16px;
  border: none;
  outline: none;
  color: black;
  padding:.41rem .5rem;
  background-color: inherit;
  font-family: inherit;
  margin: 0;
}
.zs-dropdown-content {
  display: none;
  position: absolute;
  background-color: #f9f9f9;
  min-width: 160px;
  box-shadow: 0px 8px 16px 0px rgba(0,0,0,0.2);
  z-index: 1;
}
.zs-dropdown-content > a {
  float: none;
  color: black;
  padding:.41rem .5rem;
  text-decoration: none;
  display: block;
  text-align: left;
}
.zs-dropdown-content > a:hover {
  background-color: hsl(210, 28%, 75%);
}
.zs-dropdown:hover > .zs-dropdown-content {
  display: block;
}
main {
  padding: 0 1rem;
}
article > * + * {
  margin-top: .5rem;
}
article header {
  padding: 0;
  margin: 0;
}
h1,h2,h3,h4,h5,h6 { font-family:sans-serif; font-weight:normal }
h1 { font-size:1.5rem;  margin:.65rem 0 }
h2 { font-size:1.25rem; margin:.70rem 0 }
h3 { font-size:1.15rem; margin:.75rem 0 }
h4 { font-size:1.05rem; margin;.8rem 0; font-weight: bold }
h5 { font-size:1.05rem; margin;.8rem 0 }
h6 { font-size:1.05rem; margin;.8rem 0; font-weight: lighter }
p {
  margin: .5rem 0 0 0;
}
ol,ul {
  padding-left: 1.1rem;
}
li,figure,figcaption,dl {
  margin: 0;
}
dt {
  margin: .5rem 0 0 0;
}
dt+dd {
  margin-top: 0;
}
dd {
  margin: .5rem 0 0 2rem;
}
dd > p:first-child {
  margin: 0 0 0 0;
}
blockquote {
  border-left: 0.5rem solid lightgray;
  padding-left: 1rem;
  margin-left: 1rem;
  margin-right: 2rem;
  font-style: italic;
}
blockquote p {
  margin-bottom: .5rem;
}
blockquote cite {
  font-style: normal;
}
table {
  border-collapse: collapse;
  border-spacing: 0;
  max-width: 100%;
}
th,td {
  text-align: left;
  padding: .25rem .5rem;
}
td { border-bottom: 1px solid hsl(0, 0%, 85%); }
thead th { border-bottom: 2px solid hsl(0, 0%, 70%); }
tfoot th { border-top: 2px solid hsl(0, 0%, 70%); }
main form {
  padding: 0 .5em;
  margin: .5em 0 0 0;
}
main form:after {
  content: ".";
  display: block;
  height: 0;
  clear: both;
  visibility: hidden;
}
main form div {
  margin: .5em 0 0 0
}
input {
  font-family: monospace;
}
input[type="submit"],button,select {
  font: inherit;
}
label { font-family: sans-serif; font-size:.9rem }
label::after { content:":" }
textarea {
  font-family: monospace;
  resize: vertical;
  width: 100%;
}
.zs-input {
  padding: .5em;
  display:block;
  border:none;
  border-bottom:1px solid #ccc;
  width:100%;
}
.zs-button {
  float:right;
  margin: .5em 0 .5em 1em;
}
a:not([class]) {
  text-decoration-skip-ink: auto;
}
.zs-broken {
  text-decoration: line-through;
}
img {
  max-width: 100%;
}
.zs-endnotes {
  padding-top: .5rem;
  border-top: 1px solid;
}
code,pre,kbd {
  font-family: monospace;
  font-size: 85%;
}
code {
  padding: .1rem .2rem;
  background: #f0f0f0;
  border: 1px solid #ccc;
  border-radius: .25rem;
}
pre {
  padding: .5rem .7rem;
  max-width: 100%;
  overflow: auto;
  border: 1px solid #ccc;
  border-radius: .5rem;
  background: #f0f0f0;
}
pre code {
  font-size: 95%;
  position: relative;
  padding: 0;
  border: none;
}
div.zs-indication {
  padding: .5rem .7rem;
  max-width: 100%;
  border-radius: .5rem;
  border: 1px solid black;
}
div.zs-indication p:first-child {
  margin-top: 0;
}
span.zs-indication {
  border: 1px solid black;
  border-radius: .25rem;
  padding: .1rem .2rem;
  font-size: 95%;
}
.zs-example { border-style: dotted !important }
.zs-error {
  background-color: lightpink;
  border-style: none !important;
  font-weight: bold;
}
kbd {
  background: hsl(210, 5%, 100%);
  border: 1px solid hsl(210, 5%, 70%);
  border-radius: .25rem;
  padding: .1rem .2rem;
  font-size: 75%;
}
.zs-meta {
  font-size:.75rem;
  color:#888;
  margin-bottom:1rem;
}
.zs-meta a {
  color:#888;
}
h1+.zs-meta {
  margin-top:-1rem;
}
details > summary {
  width: 100%;
  background-color: #eee;
  font-family:sans-serif;
}
details > ul {
  margin-top:0;
  padding-left:2rem;
  background-color: #eee;
}
footer {
  padding: 0 1rem;
}
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
`,
	},

	id.TOCNewTemplateZid: {
		constHeader{
			meta.KeyTitle:  "New Menu",
			meta.KeyRole:   meta.ValueRoleZettel,
			meta.KeySyntax: meta.ValueSyntaxZmk,
		},
		domain.Content(fmt.Sprintf(`This zettel lists all zettel that should act as a template for new zettel.
These zettel will be included in the ""New"" menu of the WEBUI.
* [[New Zettel|%v]]
* [[New User|%v]]`, id.TemplateNewZettelZid, id.TemplateNewUserZid)),
	},

	id.TemplateNewZettelZid: {
		constHeader{
			meta.KeyTitle:  "New Zettel",
			meta.KeyRole:   meta.ValueRoleZettel,
			meta.KeySyntax: meta.ValueSyntaxZmk,
		},
		"",
	},

	id.TemplateNewUserZid: {
		constHeader{
			meta.KeyTitle:      "New User",
			meta.KeyRole:       meta.ValueRoleUser,
			meta.KeyCredential: "",
			meta.KeyUserID:     "",
			meta.KeyUserRole:   meta.ValueUserRoleReader,
			meta.KeySyntax:     meta.ValueSyntaxNone,
		},
		"",
	},

	id.DefaultHomeZid: {
		constHeader{
			meta.KeyTitle:  "Home",
			meta.KeyRole:   meta.ValueRoleZettel,
			meta.KeySyntax: meta.ValueSyntaxZmk,
		},
		`=== Thank you for using Zettelstore!

You will find the lastest information about Zettelstore at [[https://zettelstore.de]].
Check that website regulary for [[upgrades|https://zettelstore.de/home/doc/trunk/www/download.wiki]] to the latest version.
You should consult the [[change log|https://zettelstore.de/home/doc/trunk/www/changes.wiki]] before upgrading.
Sometimes, you have to edit some of your Zettelstore-related zettel before upgrading.
Since Zettelstore is currently in a development state, every upgrade might fix some of your problems.
To check for versions, there is a zettel with the [[current version|00000000000001]] of your Zettelstore.

If you have problems concerning Zettelstore,
do not hesitate to get in [[contact with the main developer|mailto:ds@zettelstore.de]].

=== Reporting errors
If you have encountered an error, please include the content of the following zettel in your mail:
* [[Zettelstore Version|00000000000001]]
* [[Zettelstore Operating System|00000000000003]]
* [[Zettelstore Startup Configuration|00000000000096]]
* [[Zettelstore Startup Values|00000000000098]]
* [[Zettelstore Runtime Configuration|00000000000100]]

Additionally, you have to describe, what you have done before that error occurs
and what you have expected instead.
Please do not forget to include the error message, if there is one.

Some of above Zettelstore zettel can only be retrieved if you enabled ""expert mode"".
Otherwise, only some zettel are linked.
To enable expert mode, edit the zettel [[Zettelstore Runtime Configuration|00000000000100]]:
please set the metadata value of the key ''expert-mode'' to true.
To do you, enter the string ''expert-mode:true'' inside the edit view of the metadata.

=== Information about this zettel
This zettel is your home zettel.
It is part of the Zettelstore software itself.
Every time you click on the [[Home|//]] link in the menu bar, you will be redirected to this zettel.

You can change the content of this zettel by clicking on ""Edit"" above.
This allows you to customize your home zettel.

Alternatively, you can designate another zettel as your home zettel.
Edit the [[Zettelstore Runtime Configuration|00000000000100]] by adding the metadata key ''home-zettel''.
Its value is the identifier of the zettel that should act as the new home zettel.
You will find the identifier of each zettel between their ""Edit"" and the ""Info"" link above.
The identifier of this zettel is ''00010000000000''.
If you provide a wrong identifier, this zettel will be shown as the home zettel.
Take a look inside the manual for further details.
`},
}
