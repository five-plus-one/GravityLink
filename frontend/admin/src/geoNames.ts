/** ip2region 国名 → ECharts world GeoJSON `name` 的映射；未列出的英文名原样透传。 */
const COUNTRY_TO_WORLD: Record<string, string> = {
  中国: 'China',
  中国大陆: 'China',
  中国台湾: 'Taiwan',
  台湾: 'Taiwan',
  香港: 'China',
  澳门: 'China',
  美国: 'United States',
  英国: 'United Kingdom',
  韩国: 'Korea',
  南朝鲜: 'Korea',
  朝鲜: 'Dem. Rep. Korea',
  俄罗斯: 'Russia',
  日本: 'Japan',
  法国: 'France',
  德国: 'Germany',
  意大利: 'Italy',
  西班牙: 'Spain',
  葡萄牙: 'Portugal',
  荷兰: 'Netherlands',
  比利时: 'Belgium',
  瑞士: 'Switzerland',
  瑞典: 'Sweden',
  挪威: 'Norway',
  丹麦: 'Denmark',
  芬兰: 'Finland',
  波兰: 'Poland',
  奥地利: 'Austria',
  希腊: 'Greece',
  土耳其: 'Turkey',
  乌克兰: 'Ukraine',
  加拿大: 'Canada',
  澳大利亚: 'Australia',
  新西兰: 'New Zealand',
  新加坡: 'Singapore',
  马来西亚: 'Malaysia',
  泰国: 'Thailand',
  越南: 'Vietnam',
  菲律宾: 'Philippines',
  印度尼西亚: 'Indonesia',
  印度: 'India',
  巴基斯坦: 'Pakistan',
  孟加拉国: 'Bangladesh',
  阿联酋: 'United Arab Emirates',
  沙特阿拉伯: 'Saudi Arabia',
  以色列: 'Israel',
  伊朗: 'Iran',
  伊拉克: 'Iraq',
  埃及: 'Egypt',
  南非: 'South Africa',
  尼日利亚: 'Nigeria',
  肯尼亚: 'Kenya',
  巴西: 'Brazil',
  阿根廷: 'Argentina',
  智利: 'Chile',
  墨西哥: 'Mexico',
  哥伦比亚: 'Colombia',
  秘鲁: 'Peru',
};

/** ip2region 中国省名 → DataV china GeoJSON `name`。 */
const PROVINCE_TO_CHINA: Record<string, string> = {
  内蒙古: '内蒙古自治区',
  广西: '广西壮族自治区',
  西藏: '西藏自治区',
  新疆: '新疆维吾尔自治区',
  宁夏: '宁夏回族自治区',
  香港: '香港特别行政区',
  澳门: '澳门特别行政区',
  香港特别行政区: '香港特别行政区',
  澳门特别行政区: '澳门特别行政区',
  内蒙古自治区: '内蒙古自治区',
  广西壮族自治区: '广西壮族自治区',
  西藏自治区: '西藏自治区',
  新疆维吾尔自治区: '新疆维吾尔自治区',
  宁夏回族自治区: '宁夏回族自治区',
};

export function toWorldMapName(raw: string): string | null {
  const name = raw.trim();
  if (!name || name === '未知' || name === 'Reserved' || name === '0') return null;
  if (COUNTRY_TO_WORLD[name]) return COUNTRY_TO_WORLD[name];
  // 已是 GeoJSON 英文名（ip2region 境外记录）
  return name;
}

export function toChinaMapName(raw: string): string | null {
  const name = raw.trim();
  if (!name || name === '未知' || name === 'Reserved' || name === '0') return null;
  if (PROVINCE_TO_CHINA[name]) return PROVINCE_TO_CHINA[name];
  // 直接匹配 GeoJSON 全称（如「江苏省」「北京市」「台湾省」）
  return name;
}
