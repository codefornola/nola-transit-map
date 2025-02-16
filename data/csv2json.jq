#CREDIT: csv2json.jq from https://stackoverflow.com/questions/29663187/csv-to-json-using-jq 

def objectify(headers):
  def tonumberq: tonumber? // .;
  . as $in
  | reduce range(0; headers|length) as $i ({}; .[headers[$i]] = ($in[$i]) );

def csv2table:
  # For jq 1.4, replace the following line by:  def trim: .;
  def trim: sub("^ +";"") |  sub(" +$";"");
  split("\n") | map( split(",") | map(trim) );

def csv2json:
  csv2table
  | .[0] as $headers
  | reduce (.[1:][] | select(length > 0) ) as $row
      ( []; . + [ $row|objectify($headers) ]);
