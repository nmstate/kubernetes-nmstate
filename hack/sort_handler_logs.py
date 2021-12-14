#!/usr/bin/env python

import sys, getopt
import json


def help():
    print('sort_handler_logs.py -i <inputfile> [-o <outputfile>]')


def sort_log_lines(lines):
    return sorted(filter_json_lines(lines), key=get_line_timestamp)


def filter_json_lines(lines):
    return [line for line in lines if parse_line(line) is not None]


def parse_line(line):
    try:
        l = json.loads(line)
    except ValueError:
        return None
    return l


def get_line_timestamp(line):
    return parse_line(line)['ts']


if __name__ == '__main__':
    inputfile = ''
    outputfile = ''
    lines = []

    try:
        opts, args = getopt.getopt(sys.argv[1:], "hi:o:", ["ifile=", "ofile="])
    except getopt.GetoptError:
        help()
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            help()
            sys.exit()
        elif opt in ("-i", "--ifile"):
            inputfile = arg
        elif opt in ("-o", "--ofile"):
            outputfile = arg

    if outputfile == '':
        outputfile = inputfile + "_sorted"

    with open(inputfile) as f:
        lines = f.readlines()

    sorted_lines = sort_log_lines(lines)

    with open(outputfile, "w+") as of:
        for l in sorted_lines:
            of.write(l)
