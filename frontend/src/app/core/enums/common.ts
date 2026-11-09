export enum FilterOperator {
  Equals = 'Equals',
  GreaterThan = 'GreaterThan',
  LessThan = 'LessThan',
  GreaterThanOrEquals = 'GreaterThanOrEquals',
  LessThanOrEquals = 'LessThanOrEquals',
  Contains = 'Contains',
  StartsWith = 'StartsWith',
  EndsWith = 'EndsWith',
  IsNull = 'IsNull',
}

export enum TemporalPartType {
  None = 'None',
  DateOnly = 'DateOnly',
  Date = 'Date',
  Month = 'Month',
  Year = 'Year',
}
