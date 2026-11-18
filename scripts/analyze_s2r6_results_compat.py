#!/usr/bin/env python3
"""Read empty candidate receipts without changing frozen code or evidence.

The original sealed analyzer assumes trials is a list. Go emits null when a
whole opportunity is rejected before candidate generation. Interpret that as
zero candidates, retain the opportunity/rejection, and run the sealed rules.
"""
import analyze_s2r6_results as sealed

original_read=sealed.old.read
def compatible_read(path):
 value=original_read(path)
 if path.name=='receipts.json':
  for receipt in value:
   if receipt['diagnosis'].get('trials') is None:receipt['diagnosis']['trials']=[]
 return value

sealed.old.read=compatible_read
if __name__=='__main__':sealed.main()
