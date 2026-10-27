**meminfo%**__<MEMINFO_KEY><OPERATOR><VALUE>__::
  to restrict a section based on entries of _/proc/meminfo_: +
  The memory tag compares an entry of _/proc/meminfo_ with the given numerical
  value by using one of the comparison operators: *==*, *<*, *<=*, *>*, *>=*, *!=*. +
  The value uses the same unit the key in _/proc/meminfo_ already has (if any).
  For memory capacities and sizes in _/proc/meminfo_, this is usually in *kB*. +
  If the specified key is not present in _/proc/meminfo_ or the comparison condition
  is not satisfied, the section will be skipped. +
  {nbsp} +
  Syntax: +
  *meminfo%MEMINFO_KEY OPERATOR VALUE* +
  {nbsp} +
  Valid operators are: *==*, *<*, *<=*, *>*, *>=*, *!=* +
  {nbsp} +
  Examples: +
  Tune only if total system memory is at least 2 TiB (2147483648 kB): +
  [sysctl:meminfo%MemTotal>=2147483648] +
  {nbsp} +
  Tune only if available memory is greater than 1 GiB (1048576 kB): +
  [sysctl:meminfo%MemAvailable>1048576] +
  {nbsp} +
  Tune only if HugePages_Total is configured: +
  [vm:meminfo%HugePages_Total>0]
