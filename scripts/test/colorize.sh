
mygrep() {
    # 38;5;N foreground color
    # 48;5;N background color
    # GREP_COLORS="ms=$2" grep -E "^|$1" --color='always' ; 
    case "$3" in
        "bred") C="ms=101" ;;
        "bgreen") C="ms=102;1" ;;
        "byellow") C="ms=103" ;;
        "bblue") C="ms=104" ;;
        "bgrey") C="ms=40" ;;
        "red") C="ms=91" ;;
        "orange") C="ms=93" ;;
        "debug") C="ms=38;5;17" ;;
        "green") C="ms=92" ;;
        "yellow") C="ms=93" ;;
        "blue") C="ms=94" ;;
        "grey") C="ms=30" ;;
        *) echo "Invalid color specified"; exit 1 ;;
    esac

    # GREP_COLORS=$C grep -P "^|$1" --color='always'
    GREP_COLORS=$C grep $1 "^|$2" --color='always'
} 