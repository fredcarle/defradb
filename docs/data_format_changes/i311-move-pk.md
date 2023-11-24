# Move primary key outside of data scan path

Primary keys are now stored in ```db/data/pk/[CollectionId]/[DocID] => {}```.  Benchmarks suggest minor boost (10% with 4 fields).
