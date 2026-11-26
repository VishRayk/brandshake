// Must match the server's vocabularies in go/internal/profile/profile.go.

export const NICHES = [
  ['art', 'Art & design'],
  ['automotive', 'Automotive'],
  ['beauty', 'Beauty'],
  ['business', 'Business'],
  ['comedy', 'Comedy'],
  ['education', 'Education'],
  ['fashion', 'Fashion'],
  ['finance', 'Finance'],
  ['fitness', 'Fitness'],
  ['food', 'Food'],
  ['gaming', 'Gaming'],
  ['health', 'Health'],
  ['lifestyle', 'Lifestyle'],
  ['music', 'Music'],
  ['parenting', 'Parenting'],
  ['photography', 'Photography'],
  ['sports', 'Sports'],
  ['tech', 'Tech'],
  ['travel', 'Travel'],
  ['other', 'Other'],
] as const

export const INDUSTRIES = [
  ['automotive', 'Automotive'],
  ['beauty_personal_care', 'Beauty & personal care'],
  ['education', 'Education'],
  ['fashion_apparel', 'Fashion & apparel'],
  ['finance', 'Finance'],
  ['food_beverage', 'Food & beverage'],
  ['gaming', 'Gaming'],
  ['health_fitness', 'Health & fitness'],
  ['home_living', 'Home & living'],
  ['media_entertainment', 'Media & entertainment'],
  ['tech_apps', 'Tech & apps'],
  ['travel_hospitality', 'Travel & hospitality'],
  ['other', 'Other'],
] as const

export const LANGUAGES = [
  ['en', 'English'],
  ['hi', 'Hindi'],
  ['bn', 'Bengali'],
  ['gu', 'Gujarati'],
  ['kn', 'Kannada'],
  ['ml', 'Malayalam'],
  ['mr', 'Marathi'],
  ['or', 'Odia'],
  ['pa', 'Punjabi'],
  ['ta', 'Tamil'],
  ['te', 'Telugu'],
  ['ur', 'Urdu'],
] as const

export const PLATFORMS = [
  ['instagram', 'Instagram'],
  ['youtube', 'YouTube'],
  ['x', 'X'],
  ['linkedin', 'LinkedIn'],
  ['facebook', 'Facebook'],
] as const

export const STATES = [
  'Andaman and Nicobar Islands',
  'Andhra Pradesh',
  'Arunachal Pradesh',
  'Assam',
  'Bihar',
  'Chandigarh',
  'Chhattisgarh',
  'Dadra and Nagar Haveli and Daman and Diu',
  'Delhi',
  'Goa',
  'Gujarat',
  'Haryana',
  'Himachal Pradesh',
  'Jammu and Kashmir',
  'Jharkhand',
  'Karnataka',
  'Kerala',
  'Ladakh',
  'Lakshadweep',
  'Madhya Pradesh',
  'Maharashtra',
  'Manipur',
  'Meghalaya',
  'Mizoram',
  'Nagaland',
  'Odisha',
  'Puducherry',
  'Punjab',
  'Rajasthan',
  'Sikkim',
  'Tamil Nadu',
  'Telangana',
  'Tripura',
  'Uttar Pradesh',
  'Uttarakhand',
  'West Bengal',
] as const

type Pairs = readonly (readonly [string, string])[]

export function labelOf(pairs: Pairs, value: string): string {
  return pairs.find(([v]) => v === value)?.[1] ?? value
}
