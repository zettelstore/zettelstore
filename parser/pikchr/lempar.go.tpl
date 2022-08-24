/*
** 2000-05-29
**
** The author disclaims copyright to this source code.  In place of
** a legal notice, here is a blessing:
**
**    May you do good and not evil.
**    May you find forgiveness for yourself and forgive others.
**    May you share freely, never taking more than you give.
**
*************************************************************************
** Driver template for the LEMON parser generator.
**
** The "lemon" program processes an LALR(1) input grammar file, then uses
** this template to construct a parser.  The "lemon" program inserts text
** at each "%%" line.  Also, any "P-a-r-s-e" identifer prefix (without the
** interstitial "-" characters) contained in this template is changed into
** the value of the %name directive from the grammar.  Otherwise, the content
** of this template is copied straight through into the generate parser
** source file.
**
** The following is the concatenation of all %include directives from the
** input grammar file:
 */

package main

import (
	"fmt"
	"io"
	"os"
)

/************ Begin %include sections from the grammar ************************/
%%

/**************** End of %include directives **********************************/
/* These constants specify the various numeric values for terminal symbols.
***************** Begin token definitions *************************************/

%%
/**************** End token definitions ***************************************/

/* The next sections is a series of control #defines.
** various aspects of the generated parser.
**    YYCODETYPE         is the data type used to store the integer codes
**                       that represent terminal and non-terminal symbols.
**                       "unsigned char" is used if there are fewer than
**                       256 symbols.  Larger types otherwise.
**    YYNOCODE           is a number of type YYCODETYPE that is not used for
**                       any terminal or nonterminal symbol.
**    YYFALLBACK         If defined, this indicates that one or more tokens
**                       (also known as: "terminal symbols") have fall-back
**                       values which should be used if the original symbol
**                       would not parse.  This permits keywords to sometimes
**                       be used as identifiers, for example.
**    YYACTIONTYPE       is the data type used for "action codes" - numbers
**                       that indicate what to do in response to the next
**                       token.
**    ParseTOKENTYPE     is the data type used for minor type for terminal
**                       symbols.  Background: A "minor type" is a semantic
**                       value associated with a terminal or non-terminal
**                       symbols.  For example, for an "ID" terminal symbol,
**                       the minor type might be the name of the identifier.
**                       Each non-terminal can have a different minor type.
**                       Terminal symbols all have the same minor type, though.
**                       This macros defines the minor type for terminal
**                       symbols.
**    YYMINORTYPE        is the data type used for all minor types.
**                       This is typically a union of many types, one of
**                       which is ParseTOKENTYPE.  The entry in the union
**                       for terminal symbols is called "yy0".
**    YYSTACKDEPTH       is the maximum depth of the parser's stack.  If
**                       zero the stack is dynamically sized using realloc()
**    ParseARG_SDECL     A static variable declaration for the %extra_argument
**    ParseARG_PDECL     A parameter declaration for the %extra_argument
**    ParseARG_PARAM     Code to pass %extra_argument as a subroutine parameter
**    ParseARG_STORE     Code to store %extra_argument into yypParser
**    ParseARG_FETCH     Code to extract %extra_argument from yypParser
**    ParseCTX_*         As ParseARG_ except for %extra_context
**    YYERRORSYMBOL      is the code number of the error symbol.  If not
**                       defined, then do no error processing.
**    YYNSTATE           the combined number of states.
**    YYNRULE            the number of rules in the grammar
**    YYNTOKEN           Number of terminal symbols
**    YY_MAX_SHIFT       Maximum value for shift actions
**    YY_MIN_SHIFTREDUCE Minimum value for shift-reduce actions
**    YY_MAX_SHIFTREDUCE Maximum value for shift-reduce actions
**    YY_ERROR_ACTION    The yy_action[] code for syntax error
**    YY_ACCEPT_ACTION   The yy_action[] code for accept
**    YY_NO_ACTION       The yy_action[] code for no-op
**    YY_MIN_REDUCE      Minimum value for reduce actions
**    YY_MAX_REDUCE      Maximum value for reduce actions
 */
/************* Begin control #defines *****************************************/
%%

/************* End control #defines *******************************************/

/* Applications can choose to define yytestcase() in the %include section
** to a macro that can assist in verifying code coverage.  For production
** code the yytestcase() macro should be turned off.  But it is useful
** for testing.
 */

/* Next are the tables used to determine what action to take based on the
** current state and lookahead token.  These tables are used to implement
** functions that take a state number and lookahead value and return an
** action integer.
**
** Suppose the action integer is N.  Then the action is determined as
** follows
**
**   0 <= N <= YY_MAX_SHIFT             Shift N.  That is, push the lookahead
**                                      token onto the stack and goto state N.
**
**   N between YY_MIN_SHIFTREDUCE       Shift to an arbitrary state then
**     and YY_MAX_SHIFTREDUCE           reduce by rule N-YY_MIN_SHIFTREDUCE.
**
**   N == YY_ERROR_ACTION               A syntax error has occurred.
**
**   N == YY_ACCEPT_ACTION              The parser accepts its input.
**
**   N == YY_NO_ACTION                  No such action.  Denotes unused
**                                      slots in the yy_action[] table.
**
**   N between YY_MIN_REDUCE            Reduce by rule N-YY_MIN_REDUCE
**     and YY_MAX_REDUCE
**
** The action table is constructed as a single large table named yy_action[].
** Given state S and lookahead X, the action is computed as either:
**
**    (A)   N = yy_action[ yy_shift_ofst[S] + X ]
**    (B)   N = yy_default[S]
**
** The (A) formula is preferred.  The B formula is used instead if
** yy_lookahead[yy_shift_ofst[S]+X] is not equal to X.
**
** The formulas above are for computing the action when the lookahead is
** a terminal symbol.  If the lookahead is a non-terminal (as occurs after
** a reduce action) then the yy_reduce_ofst[] array is used in place of
** the yy_shift_ofst[] array.
**
** The following are the tables generated in this section:
**
**  yy_action[]        A single table containing all actions.
**  yy_lookahead[]     A table containing the lookahead for each entry in
**                     yy_action.  Used to detect hash collisions.
**  yy_shift_ofst[]    For each state, the offset into yy_action for
**                     shifting terminals.
**  yy_reduce_ofst[]   For each state, the offset into yy_action for
**                     shifting non-terminals after a reduce.
**  yy_default[]       Default action for each state.
**
*********** Begin parsing tables **********************************************/
%%

/********** End of lemon-generated parsing tables *****************************/

/* The next table maps tokens (terminal symbols) into fallback tokens.
** If a construct like the following:
**
**      %fallback ID X Y Z.
**
** appears in the grammar, then ID becomes a fallback token for X, Y,
** and Z.  Whenever one of the tokens X, Y, or Z is input to the parser
** but it does not parse, the type of the token is changed to ID and
** the parse is retried before an error is thrown.
**
** This feature can be used, for example, to cause some keywords in a language
** to revert to identifiers if they keyword does not apply in the context where
** it appears.
 */
var yyFallback = []YYCODETYPE{
	//
%%
}

/* The following structure represents a single element of the
** parser's stack.  Information stored includes:
**
**   +  The state number for the parser at this level of the stack.
**
**   +  The value of the token stored at this level of the stack.
**      (In other words, the "major" token.)
**
**   +  The semantic value stored at this level of the stack.  This is
**      the information used by the action routines in the grammar.
**      It is sometimes called the "minor" token.
**
** After the "shift" half of a SHIFTREDUCE action, the stateno field
** actually contains the reduce action for the second half of the
** SHIFTREDUCE.
 */
type yyStackEntry struct {
	stateno YYACTIONTYPE /* The state-number, or reduce action in SHIFTREDUCE */
	major   YYCODETYPE   /* The major token value.  This is the code
	 ** number for the token at this stack level */
	minor YYMINORTYPE /* The user-supplied minor token value.  This
	 ** is the value of the token  */
}

/* The state of the parser is completely contained in an instance of
** the following structure */
type yyParser struct {
	yytos int /* Index of top element on the stack */
	// #ifdef YYTRACKMAXSTACKDEPTH
	yyhwm int /* High-water mark of the stack */
	// #endif
	// #ifndef YYNOERRORRECOVERY
	yyerrcnt int /* Shifts left before out of the error */
	// #endif
	ParseARG_SDECL/* A place to hold %extra_argument */
	ParseCTX_SDECL/* A place to hold %extra_context */
	yystack []yyStackEntry
}

var yyTraceFILE *os.File
var yyTracePrompt string

/*
** Turn parser tracing on by giving a stream to which to write the trace
** and a prompt to preface each trace message.  Tracing is turned off
** by making either argument NULL
**
** Inputs:
** <ul>
** <li> A FILE* to which trace output should be written.
**      If NULL, then tracing is turned off.
** <li> A prefix string written at the beginning of every
**      line of trace output.  If NULL, then tracing is
**      turned off.
** </ul>
**
** Outputs:
** None.
 */
func ParseTrace(TraceFILE *os.File, zTracePrompt string) {
	yyTraceFILE = TraceFILE
	yyTracePrompt = zTracePrompt
	if yyTraceFILE == nil {
		yyTracePrompt = ""
	} else if yyTracePrompt == "" {
		yyTraceFILE = nil
	}
}

/* For tracing shifts, the names of all terminals and nonterminals
** are required.  The following table supplies these names */
var yyTokenName = []string{
%%
}

/* For tracing reduce actions, the names of all rules are required.
 */
var yyRuleName = []string{
%%
}

/*
** Try to increase the size of the parser stack.  Return the number
** of errors.  Return 0 on success.
*/
func (p *yyParser) yyGrowStack(){
	oldSize := len(p.yystack)
	newSize := oldSize * 2 + 100
	pNew := make([]yyStackEntry, newSize)
	copy(pNew, p.yystack)
	p.yystack = pNew

	if !NDEBUG { // #ifndef NDEBUG
    if yyTraceFILE != nil {
      fmt.Fprintf(yyTraceFILE,"%sStack grows from %d to %d entries.\n",
				yyTracePrompt, oldSize, newSize);
    }
	} // #endif
}

/* Datatype of the argument to the memory allocated passed as the
** second argument to ParseAlloc() below.  This can be changed by
** putting an appropriate #define in the %include section of the input
** grammar.
 */
// #ifndef YYMALLOCARGTYPE
// # define YYMALLOCARGTYPE size_t
// #endif

/* Initialize a new parser that has already been allocated.
 */
func (yypParser *yyParser) ParseInit(ParseCTX_PDECL) {
	ParseCTX_STORE
	if !YYNOERRORRECOVERY {
		yypParser.yyerrcnt = -1
	}
	if YYSTACKDEPTH > 0 {
		yypParser.yystack = make([]yyStackEntry, YYSTACKDEPTH)
	} else {
		yypParser.yystack = []yyStackEntry{{}}
	}
	yypParser.yytos = 0
}

/*
** This function allocates a new parser.
** The only argument is a pointer to a function which works like
** malloc.
**
** Inputs:
** A pointer to the function used to allocate memory.
**
** Outputs:
** A pointer to a parser.  This pointer is used in subsequent calls
** to Parse and ParseFree.
 */
func ParseAlloc(ParseCTX_PDECL) *yyParser {
	yypParser := &yyParser{}
	ParseCTX_STORE
	yypParser.ParseInit(ParseCTX_PARAM)
	return yypParser
}

/* The following function deletes the "minor type" or semantic value
** associated with a symbol.  The symbol can be either a terminal
** or nonterminal. "yymajor" is the symbol code, and "yypminor" is
** a pointer to the value to be deleted.  The code used to do the
** deletions is derived from the %destructor and/or %token_destructor
** directives of the input grammar.
 */
func (yypParser *yyParser) yy_destructor(
	yymajor YYCODETYPE, /* Type code for object to destroy */
	yypminor *YYMINORTYPE, /* The object to be destroyed */
) {
	ParseARG_FETCH
	ParseCTX_FETCH
	switch yymajor {
	/* Here is inserted the actions which take place when a
	 ** terminal or non-terminal is destroyed.  This can happen
	 ** when the symbol is popped from the stack during a
	 ** reduce or during error processing or when a parser is
	 ** being destroyed before it is finished parsing.
	 **
	 ** Note: during a reduce, the only symbols destroyed are those
	 ** which appear on the RHS of the rule, but which are *not* used
	 ** inside the C code.
	 */
	/********* Begin destructor definitions ***************************************/
%%
	/********* End destructor definitions *****************************************/
	default:
		break /* If no destructor action specified: do nothing */
	}
}

/*
** Pop the parser's stack once.
**
** If there is a destructor routine associated with the token which
** is popped from the stack, then call it.
 */
func (pParser *yyParser) yy_pop_parser_stack() {
	assert(pParser.yytos>0, "pParser.yytos>0")
	yytos := pParser.yystack[pParser.yytos]
	pParser.yytos--
	if !NDEBUG {
		if yyTraceFILE != nil {
			fmt.Fprintf(yyTraceFILE, "%sPopping %s\n",
				yyTracePrompt,
				yyTokenName[yytos.major])
		}
	}
	pParser.yy_destructor(yytos.major, &yytos.minor)
}

/*
** Clear all secondary memory allocations from the parser
 */
func (pParser *yyParser) ParseFinalize() {
	for pParser.yytos > 0 {
		pParser.yy_pop_parser_stack()
	}
}

/*
** Deallocate and destroy a parser.  Destructors are called for
** all stack elements before shutting the parser down.
**
** If the YYPARSEFREENEVERNULL macro exists (for example because it
** is defined in a %include section of the input grammar) then it is
** assumed that the input pointer is never NULL.
 */
func (pParser *yyParser) ParseFree() {
	pParser.ParseFinalize()
}

/*
** Return the peak depth of the stack for a parser.
 */
func (pParser *yyParser) ParseStackPeak() int {
	return pParser.yyhwm
}

/* This array of booleans keeps track of the parser statement
** coverage.  The element yycoverage[X][Y] is set when the parser
** is in state X and has a lookahead token Y.  In a well-tested
** systems, every element of this matrix should end up being set.
 */
var yycoverage = [YYNSTATE][YYNTOKEN]bool{}

/*
** Write into out a description of every state/lookahead combination that
**
**   (1)  has not been used by the parser, and
**   (2)  is not a syntax error.
**
** Return the number of missed state/lookahead combinations.
 */
func ParseCoverage(out io.Writer) int {
	nMissed := 0
	for stateno := 0; stateno < YYNSTATE; stateno++ {
		i := yy_shift_ofst[stateno]
		for iLookAhead := 0; iLookAhead < YYNTOKEN; iLookAhead++ {
			if yy_lookahead[int(i)+iLookAhead] != YYCODETYPE(iLookAhead) {
				continue
			}
			if !yycoverage[stateno][iLookAhead] {
				nMissed++
			}
			if out != nil {
				ok := "missed"
				if yycoverage[stateno][iLookAhead] {
					ok = "ok"
				}
				fmt.Fprintf(out, "State %d lookahead %s %s\n", stateno,
					yyTokenName[iLookAhead],
					ok)
			}
		}
	}
	return nMissed
}

/*
** Find the appropriate action for a parser given the terminal
** look-ahead token iLookAhead.
 */
func yy_find_shift_action(
	lookAhead YYCODETYPE, /* The look-ahead token */
	stateno YYACTIONTYPE, /* Current state number */
) YYACTIONTYPE {
	iLookAhead := int(lookAhead)

	if stateno > YY_MAX_SHIFT {
		return stateno
	}
	assert(stateno <= YY_SHIFT_COUNT, "stateno <= YY_SHIFT_COUNT")
	if YYCOVERAGE {
		yycoverage[stateno][iLookAhead] = true
	}
	for {
		i := int(yy_shift_ofst[stateno])
		assert(i >= 0, "i>=0")
		assert(i <= YY_ACTTAB_COUNT, "i<=YY_ACTTAB_COUNT")
		assert(i+YYNTOKEN <= len(yy_lookahead), "i+YYNTOKEN<=len(yy_lookahead)")
		assert(iLookAhead != YYNOCODE, "iLookAhead!=YYNOCODE")
		assert(iLookAhead < YYNTOKEN, "iLookAhead < YYNTOKEN")
		i += iLookAhead
		assert(i < len(yy_lookahead), "i<len(yy_lookahead)")
		if int(yy_lookahead[i]) != iLookAhead {
			if YYFALLBACK {
				assert(iLookAhead < len(yyFallback), "iLookAhead<len(yyfallback)")
				iFallback := int(yyFallback[iLookAhead])
				if iFallback != 0 {
					if !NDEBUG {
						if yyTraceFILE != nil {
							fmt.Fprintf(yyTraceFILE, "%sFALLBACK %s => %s\n",
								yyTracePrompt, yyTokenName[iLookAhead], yyTokenName[iFallback])
						}
					}
					assert(yyFallback[iFallback] == 0, "yyFallback[iFallback]==0") /* Fallback loop must terminate */
					iLookAhead = iFallback
					continue
				}
			}
			if YYWILDCARD > 0 {
				{
					j := i - iLookAhead + YYWILDCARD
					assert(j < len(yy_lookahead), "j < len(yy_lookahead)")
					if int(yy_lookahead[j]) == YYWILDCARD && iLookAhead > 0 {
						if !NDEBUG {
							if yyTraceFILE != nil {
								fmt.Fprintf(yyTraceFILE, "%sWILDCARD %s => %s\n",
									yyTracePrompt, yyTokenName[iLookAhead],
									yyTokenName[YYWILDCARD])
							}
						} /* NDEBUG */
						return yy_action[j]
					}
				}
			} /* YYWILDCARD */
			return yy_default[stateno]
		} else {
			assert(i >= 0 && i < len(yy_action), "i >= 0 && i < len(yy_action)")
			return yy_action[i]
		}
	}
}

/*
** Find the appropriate action for a parser given the non-terminal
** look-ahead token iLookAhead.
 */
func yy_find_reduce_action(
	stateno YYACTIONTYPE, /* Current state number */
	lookAhead YYCODETYPE, /* The look-ahead token */
) YYACTIONTYPE {
	iLookAhead := int(lookAhead)
	if YYERRORSYMBOL > 0 {
		if stateno > YY_REDUCE_COUNT {
			return yy_default[stateno]
		}
	} else {
		assert(stateno <= YY_REDUCE_COUNT, "stateno <= YY_REDUCE_COUNT")
	}
	i := int(yy_reduce_ofst[stateno])
	assert(iLookAhead != YYNOCODE, "iLookAhead != YYNOCODE")
	i += iLookAhead
	if YYERRORSYMBOL > 0 {
		if i < 0 || i >= YY_ACTTAB_COUNT || int(yy_lookahead[i]) != iLookAhead {
			return yy_default[stateno]
		}
	} else {
		assert(i >= 0 && i < YY_ACTTAB_COUNT, "i >= 0 && i < YY_ACTTAB_COUNT")
		assert(int(yy_lookahead[i]) == iLookAhead, "int(yy_lookahead[i]) == iLookAhead")
	}
	return yy_action[i]
}

/*
** The following routine is called if the stack overflows.
 */
func (yypParser *yyParser) yyStackOverflow() {
	ParseARG_FETCH
	ParseCTX_FETCH
	if !NDEBUG {
		if yyTraceFILE != nil {
			fmt.Fprintf(yyTraceFILE, "%sStack Overflow!\n", yyTracePrompt)
		}
	}
	for yypParser.yytos > 0 {
		yypParser.yy_pop_parser_stack()
	}
	/* Here code is inserted which will execute if the parser
	 ** stack every overflows */
	/******** Begin %stack_overflow code ******************************************/
%%
	/******** End %stack_overflow code ********************************************/
	ParseARG_STORE /* Suppress warning about unused %extra_argument var */
	ParseCTX_STORE
}

/*
** Print tracing information for a SHIFT action
 */
func (yypParser *yyParser) yyTraceShift(yyNewState int, zTag string) {
	if !NDEBUG {
		if yyTraceFILE != nil {
			if yyNewState < YYNSTATE {
				fmt.Fprintf(yyTraceFILE, "%s%s '%s', go to state %d\n",
					yyTracePrompt, zTag, yyTokenName[yypParser.yystack[yypParser.yytos].major],
					yyNewState)
			} else {
				fmt.Fprintf(yyTraceFILE, "%s%s '%s', pending reduce %d\n",
					yyTracePrompt, zTag, yyTokenName[yypParser.yystack[yypParser.yytos].major],
					yyNewState-YY_MIN_REDUCE)
			}
		}
	}
}

/*
** Perform a shift action.
 */
func (yypParser *yyParser) yy_shift(
	yyNewState YYACTIONTYPE, /* The new state to shift in */
	yyMajor YYCODETYPE, /* The major token to shift in */
	yyMinor ParseTOKENTYPE, /* The minor token to shift in */
) {
	yypParser.yytos++

	if YYTRACKMAXSTACKDEPTH {
		if yypParser.yytos > yypParser.yyhwm {
			yypParser.yyhwm++
			assert(yypParser.yyhwm == yypParser.yytos, "yypParser.yyhwm == yypParser.yytos")
		}
	}
	if YYSTACKDEPTH > 0 {
		if yypParser.yytos >= YYSTACKDEPTH {
			yypParser.yyStackOverflow()
			return
		}
	} else {
		if yypParser.yytos+1 >= len(yypParser.yystack) {
			yypParser.yyGrowStack()
		}
	}

	if yyNewState > YY_MAX_SHIFT {
		yyNewState += YY_MIN_REDUCE - YY_MIN_SHIFTREDUCE
	}

	yytos := &yypParser.yystack[yypParser.yytos]
	yytos.stateno = yyNewState
	yytos.major = yyMajor
	yytos.minor.yy0 = yyMinor

	yypParser.yyTraceShift(int(yyNewState), "Shift")
}

/* For rule J, yyRuleInfoLhs[J] contains the symbol on the left-hand side
** of that rule */
var yyRuleInfoLhs = []YYCODETYPE{
%%
}

/* For rule J, yyRuleInfoNRhs[J] contains the negative of the number
** of symbols on the right-hand side of that rule. */
var yyRuleInfoNRhs = []int8{
%%
}

/*
** Perform a reduce action and the shift that must immediately
** follow the reduce.
**
** The yyLookahead and yyLookaheadToken parameters provide reduce actions
** access to the lookahead token (if any).  The yyLookahead will be YYNOCODE
** if the lookahead token has already been consumed.  As this procedure is
** only called from one place, optimizing compilers will in-line it, which
** means that the extra parameters have no performance impact.
 */
func (yypParser *yyParser) yy_reduce(
	yyruleno YYACTIONTYPE, /* Number of the rule by which to reduce */
	yyLookahead YYCODETYPE, /* Lookahead token, or YYNOCODE if none */
	yyLookaheadToken ParseTOKENTYPE, /* Value of the lookahead token */
	ParseCTX_PDECL/* %extra_context */) YYACTIONTYPE {
	var (
		yygoto YYCODETYPE    /* The next state */
		yyact  YYACTIONTYPE  /* The next action */
		yymsp int            /* The top of the parser's stack */
		yysize int           /* Amount to pop the stack */
		yylhsminor YYMINORTYPE
	)
	yymsp = yypParser.yytos
	_ = yylhsminor

	ParseARG_FETCH

	switch yyruleno {
	/* Beginning here are the reduction cases.  A typical example
	 ** follows:
	 **   case 0:
	 **  #line <lineno> <grammarfile>
	 **     { ... }           // User supplied code
	 **  #line <lineno> <thisfile>
	 **     break;
	 */
	/********** Begin reduce actions **********************************************/
%%
		/********** End reduce actions ************************************************/
	}
	assert(int(yyruleno) < len(yyRuleInfoLhs), "yyruleno < len(yyRuleInfoLhs)")
	yygoto = yyRuleInfoLhs[yyruleno]
	yysize = int(yyRuleInfoNRhs[yyruleno])
	yyact = yy_find_reduce_action(yypParser.yystack[yymsp+yysize].stateno, yygoto)

	/* There are no SHIFTREDUCE actions on nonterminals because the table
	 ** generator has simplified them to pure REDUCE actions. */
	assert(!(yyact > YY_MAX_SHIFT && yyact <= YY_MAX_SHIFTREDUCE),
		"!(yyact > YY_MAX_SHIFT && yyact <= YY_MAX_SHIFTREDUCE)")

	/* It is not possible for a REDUCE to be followed by an error */
	assert(yyact != YY_ERROR_ACTION, "yyact != YY_ERROR_ACTION")

	yymsp += yysize+1
	yypParser.yytos = yymsp
	yypParser.yystack[yymsp].stateno = yyact
	yypParser.yystack[yymsp].major = yygoto
	yypParser.yyTraceShift(int(yyact), "... then shift")
	return yyact
}

/*
** The following code executes when the parse fails
 */
func (yypParser *yyParser) yy_parse_failed() {
	ParseARG_FETCH
	ParseCTX_FETCH
	if !NDEBUG {
		if yyTraceFILE != nil {
			fmt.Fprintf(yyTraceFILE, "%sFail!\n", yyTracePrompt)
		}
	}
	for yypParser.yytos > 0 {
		yypParser.yy_pop_parser_stack()
	}
	/* Here code is inserted which will be executed whenever the
	 ** parser fails */
	/************ Begin %parse_failure code ***************************************/
%%

	/************ End %parse_failure code *****************************************/
	ParseARG_STORE /* Suppress warning about unused %extra_argument variable */
	ParseCTX_STORE
}

/*
** The following code executes when a syntax error first occurs.
 */
func (yypParser *yyParser) yy_syntax_error(
	yymajor YYCODETYPE, /* The major type of the error token */
	yyminor ParseTOKENTYPE, /* The minor type of the error token */
) {
	ParseARG_FETCH
	ParseCTX_FETCH
	TOKEN := yyminor
	_ = TOKEN
	/************ Begin %syntax_error code ****************************************/
%%

	/************ End %syntax_error code ******************************************/
	ParseARG_STORE /* Suppress warning about unused %extra_argument variable */
	ParseCTX_STORE
}

/*
** The following is executed when the parser accepts
 */
func (yypParser *yyParser) yy_accept() {
	ParseARG_FETCH
	ParseCTX_FETCH
	if !NDEBUG {
		if yyTraceFILE != nil {
			fmt.Fprintf(yyTraceFILE, "%sAccept!\n", yyTracePrompt)
		}
	}
	if !YYNOERRORRECOVERY {
		yypParser.yyerrcnt = -1
	}
	assert(yypParser.yytos==0, fmt.Sprintf("want yypParser.yytos == 0; got %d", yypParser.yytos))
	/* Here code is inserted which will be executed whenever the
	 ** parser accepts */
	/*********** Begin %parse_accept code *****************************************/
%%

	/*********** End %parse_accept code *******************************************/
	ParseARG_STORE /* Suppress warning about unused %extra_argument variable */
	ParseCTX_STORE
}

/* The main parser program.
** The first argument is a pointer to a structure obtained from
** "ParseAlloc" which describes the current state of the parser.
** The second argument is the major token number.  The third is
** the minor token.  The fourth optional argument is whatever the
** user wants (and specified in the grammar) and is available for
** use by the action routines.
**
** Inputs:
** <ul>
** <li> A pointer to the parser (an opaque structure.)
** <li> The major token number.
** <li> The minor token number.
** <li> An option argument of a grammar-specified type.
** </ul>
**
** Outputs:
** None.
 */
func (yypParser *yyParser) Parse(
	yymajor YYCODETYPE, /* The major token code number */
	yyminor ParseTOKENTYPE, /* The value for the token */
	/* Optional %extra_argument parameter */
) {
	var (
		yyminorunion YYMINORTYPE
		yyact        YYACTIONTYPE /* The parser action. */
		yyendofinput bool         /* True if we are at the end of input */
		yyerrorhit   bool         /* True if yymajor has invoked an error */
	)

	ParseCTX_FETCH
	ParseARG_STORE

	assert(yypParser.yystack != nil, "yypParser.yystack != nil")
	if YYERRORSYMBOL == 0 && !YYNOERRORRECOVERY {
		yyendofinput = (yymajor == 0)
	}

	yyact = yypParser.yystack[yypParser.yytos].stateno
	if !NDEBUG {
		if yyTraceFILE != nil {
			if yyact < YY_MIN_REDUCE {
				fmt.Fprintf(yyTraceFILE, "%sInput '%s' in state %d\n",
					yyTracePrompt, yyTokenName[yymajor], yyact)
			} else {
				fmt.Fprintf(yyTraceFILE, "%sInput '%s' with pending reduce %d\n",
					yyTracePrompt, yyTokenName[yymajor], yyact-YY_MIN_REDUCE)
			}
		}
	}

	for { /* Exit by "break" */
		assert(yypParser.yytos >= 0, "yypParser.yytos >= 0")
		assert(yyact == yypParser.yystack[yypParser.yytos].stateno, "yyact == yypParser.yystack[yypParser.yytos].stateno")
		yyact = yy_find_shift_action(yymajor, yyact)
		if yyact >= YY_MIN_REDUCE {
			yyruleno := yyact - YY_MIN_REDUCE /* Reduce by this rule */
			if !NDEBUG {
				assert(int(yyruleno) < len(yyRuleName), "int(yyruleno) < len(yyRuleName)")
				if yyTraceFILE != nil {
					yysize := yyRuleInfoNRhs[yyruleno]
					wea := " without external action"
					if yyruleno < YYNRULE_WITH_ACTION {
						wea = ""
					}
					if yysize != 0 {
						fmt.Fprintf(yyTraceFILE, "%sReduce %d [%s]%s, pop back to state %d.\n",
							yyTracePrompt,
							yyruleno, yyRuleName[yyruleno],
							wea,
							yypParser.yystack[yypParser.yytos+int(yysize)].stateno)
					} else {
						fmt.Fprintf(yyTraceFILE, "%sReduce %d [%s]%s.\n",
							yyTracePrompt, yyruleno, yyRuleName[yyruleno],
							wea)
					}
				}
			} /* NDEBUG */

			/* Check that the stack is large enough to grow by a single entry
			 ** if the RHS of the rule is empty.  This ensures that there is room
			 ** enough on the stack to push the LHS value */
			if yyRuleInfoNRhs[yyruleno] == 0 {
				if YYTRACKMAXSTACKDEPTH {
					if yypParser.yytos > yypParser.yyhwm {
						yypParser.yyhwm++
						assert(yypParser.yyhwm == yypParser.yytos, "yypParser.yyhwm == yypParser.yytos")
					}
				}
				if YYSTACKDEPTH > 0 {
					if yypParser.yytos >= YYSTACKDEPTH-1 {
						yypParser.yyStackOverflow()
						break
					}
				} else {
					if yypParser.yytos+1 >= len(yypParser.yystack)-1 {
						yypParser.yyGrowStack()
					}
				}
			}
			yyact = yypParser.yy_reduce(yyruleno, yymajor, yyminor,
			ParseCTX_PARAM)
		} else if yyact <= YY_MAX_SHIFTREDUCE {
			yypParser.yy_shift(yyact, yymajor, yyminor)
			if !YYNOERRORRECOVERY {
				yypParser.yyerrcnt--
			}
			break
		} else if yyact == YY_ACCEPT_ACTION {
			yypParser.yytos--
			yypParser.yy_accept()
			return
		} else {
			assert(yyact == YY_ERROR_ACTION, "yyact == YY_ERROR_ACTION")
			yyminorunion.yy0 = yyminor

			if !NDEBUG {
				if yyTraceFILE != nil {
					fmt.Fprintf(yyTraceFILE, "%sSyntax Error!\n", yyTracePrompt)
				}
			}
			if YYERRORSYMBOL > 0 {
				/* A syntax error has occurred.
				 ** The response to an error depends upon whether or not the
				 ** grammar defines an error token "ERROR".
				 **
				 ** This is what we do if the grammar does define ERROR:
				 **
				 **  * Call the %syntax_error function.
				 **
				 **  * Begin popping the stack until we enter a state where
				 **    it is legal to shift the error symbol, then shift
				 **    the error symbol.
				 **
				 **  * Set the error count to three.
				 **
				 **  * Begin accepting and shifting new tokens.  No new error
				 **    processing will occur until three tokens have been
				 **    shifted successfully.
				 **
				 */
				if yypParser.yyerrcnt < 0 {
					yypParser.yy_syntax_error(yymajor, yyminor)
				}
				yymx := yypParser.yystack[yypParser.yytos].major
				if int(yymx) == YYERRORSYMBOL || yyerrorhit {
					if !NDEBUG {
						if yyTraceFILE != nil {
							fmt.Fprintf(yyTraceFILE, "%sDiscard input token %s\n",
								yyTracePrompt, yyTokenName[yymajor])
						}
					}
					yypParser.yy_destructor(yymajor, &yyminorunion)
					yymajor = YYNOCODE
				} else {
					for yypParser.yytos > 0 {
						yyact = yy_find_reduce_action(yypParser.yystack[yypParser.yytos].stateno,
							YYERRORSYMBOL)
						if yyact <= YY_MAX_SHIFTREDUCE {
							break
						}
						yypParser.yy_pop_parser_stack()
					}
					if yypParser.yytos <= 0 || yymajor == 0 {
						yypParser.yy_destructor(yymajor, &yyminorunion)
						yypParser.yy_parse_failed()
						if !YYNOERRORRECOVERY {
							yypParser.yyerrcnt = -1
						}
						yymajor = YYNOCODE
					} else if yymx != YYERRORSYMBOL {
						yypParser.yy_shift(yyact, YYERRORSYMBOL, yyminor)
					}
				}
				yypParser.yyerrcnt = 3
				yyerrorhit = true
				if yymajor == YYNOCODE {
					break
				}
				yyact = yypParser.yystack[yypParser.yytos].stateno
			} else if YYNOERRORRECOVERY {
				/* If the YYNOERRORRECOVERY macro is defined, then do not attempt to
				 ** do any kind of error recovery.  Instead, simply invoke the syntax
				 ** error routine and continue going as if nothing had happened.
				 **
				 ** Applications can set this macro (for example inside %include) if
				 ** they intend to abandon the parse upon the first syntax error seen.
				 */
				yypParser.yy_syntax_error(yymajor, yyminor)
				yypParser.yy_destructor(yymajor, &yyminorunion)
				break
			} else { /* YYERRORSYMBOL is not defined */
				/* This is what we do if the grammar does not define ERROR:
				 **
				 **  * Report an error message, and throw away the input token.
				 **
				 **  * If the input token is $, then fail the parse.
				 **
				 ** As before, subsequent error messages are suppressed until
				 ** three input tokens have been successfully shifted.
				 */
				if yypParser.yyerrcnt <= 0 {
					yypParser.yy_syntax_error(yymajor, yyminor)
				}
				yypParser.yyerrcnt = 3
				yypParser.yy_destructor(yymajor, &yyminorunion)
				if yyendofinput {
					yypParser.yy_parse_failed()
					if !YYNOERRORRECOVERY {
						yypParser.yyerrcnt = -1
					}
				}
				break
			}
		}
	}
	if !NDEBUG {
		if yyTraceFILE != nil {
			cDiv := '['
			fmt.Fprintf(yyTraceFILE, "%sReturn. Stack=", yyTracePrompt)
			for _, i := range yypParser.yystack[1:yypParser.yytos+1] {
				fmt.Fprintf(yyTraceFILE, "%c%s", cDiv, yyTokenName[i.major])
				cDiv = ' '
			}
			fmt.Fprintf(yyTraceFILE, "]\n")
		}
	}
	return
}

/*
** Return the fallback token corresponding to canonical token iToken, or
** 0 if iToken has no fallback.
 */
func ParseFallback(iToken int) YYCODETYPE {
	if YYFALLBACK {
		assert(iToken < len(yyFallback), "iToken < len(yyFallback)")
		return yyFallback[iToken]
	} else {
		return 0
	}
}

// assert is used in various places in the generated and template code
// to check invariants.
func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}
